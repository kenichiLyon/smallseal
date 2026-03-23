package adapters

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/sealdice/smallseal/dice/types"
)

const actionTimeout = 15 * time.Second

type sessionRole string

const (
	roleUnknown sessionRole = ""
	roleEvent   sessionRole = "event"
	roleAPI     sessionRole = "api"
	roleUnified sessionRole = "unified"
)

// PlatformAdapterOB11 bridges Dice with a OneBot11 endpoint over WebSocket (forward or reverse).
type PlatformAdapterOB11 struct {
	WSReverseURL  string `json:"ws_reverse" yaml:"ws_reverse"`
	WSForwardAddr string `json:"ws_forward" yaml:"ws_forward"`
	AccessToken   string `json:"access_token" yaml:"access_token"`
	Secret        string `json:"secret" yaml:"secret"`

	callback AdapterCallback

	running atomic.Bool

	apiSession   atomic.Pointer[ob11Session]
	eventSession atomic.Pointer[ob11Session]

	requestSeq atomic.Uint64
	echoWaiter ob11EchoWaiter

	lifecycleMu sync.Mutex
	serveCancel context.CancelFunc
	dispatcher  *ob11EventDispatcher

	forwardServer *http.Server
}

// ob11Session wraps a websocket connection with role metadata.
type ob11Session struct {
	conn      *websocket.Conn
	role      sessionRole
	writeMu   sync.Mutex
	closeOnce sync.Once
}

func newOB11Session(conn *websocket.Conn, role sessionRole) *ob11Session {
	return &ob11Session{conn: conn, role: role}
}

func (s *ob11Session) writeJSON(v any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.WriteJSON(v)
}

func (s *ob11Session) close() {
	s.closeOnce.Do(func() {
		_ = s.conn.Close()
	})
}

// SetCallback registers adapter callbacks.
func (pa *PlatformAdapterOB11) SetCallback(callback AdapterCallback) {
	pa.callback = callback
}

// IsAlive reports whether at least one active event session exists.
func (pa *PlatformAdapterOB11) IsAlive() bool {
	return pa.running.Load()
}

// Serve boots reverse and/or forward websocket endpoints.
func (pa *PlatformAdapterOB11) Serve(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	if pa.WSReverseURL == "" && pa.WSForwardAddr == "" {
		zap.S().Named("adapter").Warn("ob11 adapter: neither ws_reverse nor ws_forward configured")
		return
	}

	runCtx, cancel := context.WithCancel(ctx)
	pa.lifecycleMu.Lock()
	if pa.serveCancel != nil {
		pa.serveCancel()
	}
	pa.serveCancel = cancel
	oldDispatcher := pa.dispatcher
	pa.dispatcher = newOB11EventDispatcher(runCtx, ob11DispatchWorkerCount, ob11DispatchQueueSize, func(job ob11DispatchJob) {
		if err := pa.handleDispatchedFrame(job.postType, job.payload); err != nil {
			zap.S().Named("adapter").Debugf("ob11 dispatched frame failed: %v", err)
		}
	})
	pa.lifecycleMu.Unlock()
	oldDispatcher.wait()

	if pa.WSReverseURL != "" {
		go pa.loopReverse(runCtx)
	}

	if pa.WSForwardAddr != "" {
		go pa.listenForward(runCtx)
	}
}

// Close shuts down active sessions and forward server if any.
func (pa *PlatformAdapterOB11) Close() {
	pa.lifecycleMu.Lock()
	cancel := pa.serveCancel
	pa.serveCancel = nil
	dispatcher := pa.dispatcher
	pa.dispatcher = nil
	pa.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if dispatcher != nil {
		dispatcher.wait()
	}

	if srv := pa.forwardServer; srv != nil {
		_ = srv.Shutdown(context.Background())
	}

	api := pa.apiSession.Swap(nil)
	event := pa.eventSession.Swap(nil)

	if api != nil {
		pa.failPending(errors.New("ob11 adapter closed"))
		api.close()
	}
	if event != nil && event != api {
		event.close()
	}

	pa.running.Store(false)
}

func (pa *PlatformAdapterOB11) loopReverse(ctx context.Context) {
	log := zap.S().Named("adapter")
	backoff := time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		connected, err := pa.connectReverse(ctx)
		if connected {
			backoff = time.Second
		}
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Warnf("ob11 reverse ws connect failed: %v", err)
			}
		}

		if ctx.Err() != nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		if !connected && backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (pa *PlatformAdapterOB11) connectReverse(ctx context.Context) (connected bool, retErr error) {
	header := http.Header{}
	if pa.AccessToken != "" {
		header.Set("Authorization", "Bearer "+pa.AccessToken)
	}
	if pa.Secret != "" {
		header.Set("X-Self-Secret", pa.Secret)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, pa.WSReverseURL, header)
	if err != nil {
		return false, err
	}
	connected = true
	session := newOB11Session(conn, roleUnified)
	pa.setSession(session)
	defer func() {
		pa.clearSession(session, retErr)
	}()

	retErr = pa.consumeSession(ctx, session)
	return connected, retErr
}

func (pa *PlatformAdapterOB11) listenForward(ctx context.Context) {
	log := zap.S().Named("adapter")
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if ctx.Err() != nil {
			http.Error(w, "adapter shutting down", http.StatusServiceUnavailable)
			return
		}

		if pa.AccessToken != "" {
			token := r.Header.Get("Authorization")
			token = strings.TrimPrefix(token, "Bearer ")
			if token == "" {
				token = r.URL.Query().Get("access_token")
			}
			if token != pa.AccessToken {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		role := determineRole(r)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Warnf("ob11 forward ws upgrade failed: %v", err)
			return
		}

		session := newOB11Session(conn, role)
		pa.setSession(session)

		consumeErr := pa.consumeSession(ctx, session)
		pa.clearSession(session, consumeErr)
		if consumeErr != nil && !errors.Is(consumeErr, context.Canceled) {
			log.Debugf("ob11 forward ws closed: %v", consumeErr)
		}
	})

	server := &http.Server{
		Addr:    pa.WSForwardAddr,
		Handler: mux,
	}
	pa.forwardServer = server

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Errorf("ob11 forward ws listen failed: %v", err)
	}
}

func determineRole(r *http.Request) sessionRole {
	role := strings.ToLower(r.Header.Get("X-Client-Role"))
	switch role {
	case "event":
		return roleEvent
	case "api":
		return roleAPI
	}

	path := strings.ToLower(r.URL.Path)
	switch {
	case strings.Contains(path, "api"):
		return roleAPI
	case strings.Contains(path, "event"):
		return roleEvent
	default:
		return roleEvent
	}
}

func (pa *PlatformAdapterOB11) consumeSession(ctx context.Context, session *ob11Session) error {
	log := zap.S().Named("adapter")

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, payload, err := session.conn.ReadMessage()
		if err != nil {
			return err
		}

		if err := pa.dispatchFrame(ctx, payload); err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			log.Debugf("ob11 frame dispatch failed: %v", err)
		}
	}
}

func (pa *PlatformAdapterOB11) dispatchFrame(ctx context.Context, payload []byte) error {
	log := zap.S().Named("adapter")

	var base ob11BaseFrame
	if err := json.Unmarshal(payload, &base); err != nil {
		return err
	}

	if len(base.Echo) != 0 {
		if !pa.echoWaiter.resolve(base.Echo, payload) {
			log.Debugf("ob11: dropped echo response echo=%s", sanitizeRawMessage(base.Echo))
		}
		return nil
	}

	if base.PostType == "" {
		log.Debug("ob11: ignore frame without post_type and echo")
		return nil
	}

	pa.lifecycleMu.Lock()
	dispatcher := pa.dispatcher
	pa.lifecycleMu.Unlock()
	if dispatcher == nil {
		return errors.New("ob11 adapter: dispatcher not initialized")
	}
	return dispatcher.submit(ctx, ob11DispatchJob{postType: base.PostType, payload: payload})
}

func (pa *PlatformAdapterOB11) handleDispatchedFrame(postType string, payload []byte) error {
	log := zap.S().Named("adapter")

	if postType != "message" {
		if pa.callback != nil {
			evt, err := pa.convertFrameToEvent(postType, payload)
			if err != nil {
				log.Debugf("ob11 event dispatch failed: %v", err)
				pa.callback.OnError(err)
				return nil
			}

			if evt != nil {
				pa.callback.OnEvent(evt)
			}
		}
		return nil
	}

	var evt ob11EventEnvelope
	if err := json.Unmarshal(payload, &evt); err != nil {
		return err
	}

	msg := pa.convertEventToMessage(&evt)
	if msg == nil || pa.callback == nil {
		return nil
	}

	pa.callback.OnMessageReceived(&MessageSendCallbackInfo{
		Sender: &SimpleUserInfo{
			UserId:   msg.Sender.UserID,
			UserName: msg.Sender.Nickname,
		},
		Message: msg,
	})

	return nil
}

func (pa *PlatformAdapterOB11) convertFrameToEvent(postType string, payload []byte) (*types.AdapterEvent, error) {
	evt := &types.AdapterEvent{
		Platform: "QQ",
		PostType: postType,
	}

	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err == nil {
		evt.Raw = raw
	}

	switch postType {
	case "notice":
		var notice ob11NoticeEvent
		if err := json.Unmarshal(payload, &notice); err != nil {
			return nil, err
		}

		evt.Type = notice.NoticeType
		evt.SubType = notice.SubType
		evt.Time = notice.Time

		if gid := sanitizeRawMessage(notice.GroupID); gid != "" {
			evt.GroupID = FormatDiceIDQQGroup(gid)
		}
		if guild := sanitizeRawMessage(notice.GuildID); guild != "" {
			evt.GuildID = guild
		}
		if channel := sanitizeRawMessage(notice.ChannelID); channel != "" {
			evt.ChannelID = channel
		}
		if uid := sanitizeRawMessage(notice.UserID); uid != "" {
			evt.UserID = FormatDiceIDQQ(uid)
		} else if target := sanitizeRawMessage(notice.TargetID); target != "" {
			evt.UserID = FormatDiceIDQQ(target)
		}
		if op := sanitizeRawMessage(notice.OperatorID); op != "" {
			evt.OperatorID = FormatDiceIDQQ(op)
		}

	case "request":
		var req ob11RequestEvent
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, err
		}

		evt.Type = req.RequestType
		evt.SubType = req.SubType
		evt.Time = req.Time

		if gid := sanitizeRawMessage(req.GroupID); gid != "" {
			evt.GroupID = FormatDiceIDQQGroup(gid)
		}
		if uid := sanitizeRawMessage(req.UserID); uid != "" {
			evt.UserID = FormatDiceIDQQ(uid)
		}
		if op := sanitizeRawMessage(req.OperatorID); op != "" {
			evt.OperatorID = FormatDiceIDQQ(op)
		} else if inv := sanitizeRawMessage(req.InvitorID); inv != "" {
			evt.OperatorID = FormatDiceIDQQ(inv)
		}
		if guild := sanitizeRawMessage(req.GuildID); guild != "" {
			evt.GuildID = guild
		}
		if channel := sanitizeRawMessage(req.ChannelID); channel != "" {
			evt.ChannelID = channel
		}

	case "meta_event":
		var meta ob11MetaEvent
		if err := json.Unmarshal(payload, &meta); err != nil {
			return nil, err
		}

		evt.Type = meta.MetaEventType
		evt.SubType = meta.SubType
		evt.Time = meta.Time
	default:
		// pass through other post types with raw payload
	}

	if evt.Type == "" {
		evt.Type = postType
	}

	return evt, nil
}

func (pa *PlatformAdapterOB11) setSession(session *ob11Session) {
	switch session.role {
	case roleAPI:
		if old := pa.apiSession.Swap(session); old != nil && old != session {
			old.close()
		}
	case roleEvent:
		if old := pa.eventSession.Swap(session); old != nil && old != session {
			old.close()
		}
	case roleUnified:
		if old := pa.apiSession.Swap(session); old != nil && old != session {
			old.close()
		}
		if old := pa.eventSession.Swap(session); old != nil && old != session {
			old.close()
		}
	default:
		if old := pa.eventSession.Swap(session); old != nil && old != session {
			old.close()
		}
	}

	pa.running.Store(true)
}

func (pa *PlatformAdapterOB11) clearSession(session *ob11Session, cause error) {
	apiCleared := false
	if pa.apiSession.Load() == session {
		apiCleared = pa.apiSession.CompareAndSwap(session, nil)
	}
	if pa.eventSession.Load() == session {
		pa.eventSession.CompareAndSwap(session, nil)
	}

	session.close()

	if apiCleared {
		if cause == nil {
			cause = errors.New("ob11 api websocket closed")
		}
		pa.failPending(cause)
	}

	if pa.apiSession.Load() == nil && pa.eventSession.Load() == nil {
		pa.running.Store(false)
	}
}

func (pa *PlatformAdapterOB11) failPending(err error) {
	pa.echoWaiter.failAll(err)
}

func (pa *PlatformAdapterOB11) callAction(ctx context.Context, action string, params map[string]any, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}

	session := pa.getAPISession()
	if session == nil {
		return errors.New("ob11 adapter: no active API session")
	}

	// 生成随机值并转换为高进制
	randomValue, _ := rand.Int(rand.Reader, big.NewInt(1<<32)) // 32位随机数
	randomStr := randomValue.Text(62)                          // 转换为62进制（最高进制）
	echo := fmt.Sprintf("seal-%s-%d", randomStr, pa.requestSeq.Add(1))
	respCh := pa.echoWaiter.register(echo)

	payload := map[string]any{
		"action": action,
		"params": params,
		"echo":   echo,
	}

	if err := session.writeJSON(payload); err != nil {
		pa.echoWaiter.cancel(echo)
		return err
	}

	select {
	case <-ctx.Done():
		pa.echoWaiter.cancel(echo)
		return ctx.Err()
	case <-time.After(actionTimeout):
		pa.echoWaiter.cancel(echo)
		return errors.New("ob11 adapter: action timeout")
	case resp := <-respCh:

		if resp.Status != "ok" {
			reason := strings.TrimSpace(resp.Message)
			wording := strings.TrimSpace(resp.Wording)
			if reason == "" {
				reason = wording
			} else if wording != "" && wording != reason {
				reason = reason + ": " + wording
			}
			if reason != "" {
				return fmt.Errorf("ob11 adapter: %s", reason)
			}
			return fmt.Errorf("ob11 adapter: retcode=%d", resp.RetCode)
		}
		if out != nil && len(resp.Data) > 0 {
			if err := json.Unmarshal(resp.Data, out); err != nil {
				return err
			}
		}
		return nil
	}
}

func (pa *PlatformAdapterOB11) getAPISession() *ob11Session {
	if ses := pa.apiSession.Load(); ses != nil {
		return ses
	}
	if ses := pa.eventSession.Load(); ses != nil && ses.role == roleUnified {
		return ses
	}
	return nil
}

// MsgSendToGroup sends a message to a group.
func (pa *PlatformAdapterOB11) MsgSendToGroup(request *MessageSendRequest) (bool, error) {
	gid, err := parseInt64(ExtractQQGroupID(formatAnyID(request.TargetId)))
	if err != nil {
		return false, err
	}

	targetID := FormatDiceIDQQGroup(strconv.FormatInt(gid, 10))
	sentSegments, rawID, err := pa.sendSegmentsWithPoke(context.Background(), ob11SendConfig{
		messageAction: "send_group_msg",
		pokeAction:    "group_poke",
		targetParam:   "group_id",
		targetID:      gid,
	}, request.Segments)
	if len(sentSegments) > 0 {
		pa.emitEcho(sentSegments, request.Sender, "group", targetID, rawID)
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// MsgSendToPerson sends a private message.
func (pa *PlatformAdapterOB11) MsgSendToPerson(request *MessageSendRequest) (bool, error) {
	uid, err := parseInt64(ExtractQQUserID(formatAnyID(request.TargetId)))
	if err != nil {
		return false, err
	}

	targetID := FormatDiceIDQQ(strconv.FormatInt(uid, 10))
	sentSegments, rawID, err := pa.sendSegmentsWithPoke(context.Background(), ob11SendConfig{
		messageAction: "send_private_msg",
		pokeAction:    "friend_poke",
		targetParam:   "user_id",
		targetID:      uid,
	}, request.Segments)
	if len(sentSegments) > 0 {
		pa.emitEcho(sentSegments, request.Sender, "private", targetID, rawID)
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// MsgSendFileToPerson is not supported yet.
func (pa *PlatformAdapterOB11) MsgSendFileToPerson(_ *MessageSendFileRequest) (bool, error) {
	return false, errors.New("ob11 adapter: MsgSendFileToPerson not supported")
}

// MsgSendFileToGroup is not supported yet.
func (pa *PlatformAdapterOB11) MsgSendFileToGroup(_ *MessageSendFileRequest) (bool, error) {
	return false, errors.New("ob11 adapter: MsgSendFileToGroup not supported")
}

// MsgEdit is not available for OneBot 11.
func (pa *PlatformAdapterOB11) MsgEdit(_ *MessageOperationRequest) (bool, error) {
	return false, errors.New("ob11 adapter: message edit not supported")
}

// MsgRecall recalls a message by ID.
func (pa *PlatformAdapterOB11) MsgRecall(request *MessageOperationRequest) (bool, error) {
	mid, err := parseInt64(formatAnyID(request.MessageID))
	if err != nil {
		return false, err
	}

	params := map[string]any{"message_id": mid}
	if err := pa.callAction(context.Background(), "delete_msg", params, nil); err != nil {
		return false, err
	}
	return true, nil
}

// GroupMemberBan mutes a member in a group.
func (pa *PlatformAdapterOB11) GroupMemberBan(request *GroupOperationBanRequest) (bool, error) {
	gid, err := parseInt64(ExtractQQGroupID(formatAnyID(request.GroupID)))
	if err != nil {
		return false, err
	}
	uid, err := parseInt64(ExtractQQUserID(formatAnyID(request.UserID)))
	if err != nil {
		return false, err
	}

	params := map[string]any{
		"group_id": gid,
		"user_id":  uid,
		"duration": request.Duration,
	}

	if err := pa.callAction(context.Background(), "set_group_ban", params, nil); err != nil {
		return false, err
	}
	return true, nil
}

// GroupMemberKick removes a member from the group.
func (pa *PlatformAdapterOB11) GroupMemberKick(request *GroupOperationKickRequest) (bool, error) {
	gid, err := parseInt64(ExtractQQGroupID(formatAnyID(request.GroupID)))
	if err != nil {
		return false, err
	}
	uid, err := parseInt64(ExtractQQUserID(formatAnyID(request.UserID)))
	if err != nil {
		return false, err
	}

	params := map[string]any{
		"group_id":           gid,
		"user_id":            uid,
		"reject_add_request": true,
	}

	if err := pa.callAction(context.Background(), "set_group_kick", params, nil); err != nil {
		return false, err
	}
	return true, nil
}

// GroupQuit lets the bot leave a group.
func (pa *PlatformAdapterOB11) GroupQuit(request *GroupOperationQuitRequest) (bool, error) {
	gid, err := parseInt64(ExtractQQGroupID(formatAnyID(request.GroupID)))
	if err != nil {
		return false, err
	}

	params := map[string]any{
		"group_id":   gid,
		"is_dismiss": false,
	}

	if err := pa.callAction(context.Background(), "set_group_leave", params, nil); err != nil {
		return false, err
	}
	return true, nil
}

// GroupCardNameSet sets the card for a member.
func (pa *PlatformAdapterOB11) GroupCardNameSet(request *GroupOperationCardNameSetRequest) (bool, error) {
	gid, err := parseInt64(ExtractQQGroupID(formatAnyID(request.GroupID)))
	if err != nil {
		return false, err
	}
	uid, err := parseInt64(ExtractQQUserID(formatAnyID(request.UserID)))
	if err != nil {
		return false, err
	}

	params := map[string]any{
		"group_id": gid,
		"user_id":  uid,
		"card":     request.Name,
	}

	if err := pa.callAction(context.Background(), "set_group_card", params, nil); err != nil {
		return false, err
	}
	return true, nil
}

// GroupInfoGet fetches meta information about a group.
func (pa *PlatformAdapterOB11) GroupInfoGet(groupID any) (*GroupInfo, error) {
	gid, err := parseInt64(ExtractQQGroupID(formatAnyID(groupID)))
	if err != nil {
		return nil, err
	}

	params := map[string]any{
		"group_id": gid,
	}

	var resp struct {
		GroupID        int64  `json:"group_id"`
		GroupName      string `json:"group_name"`
		MemberCount    int64  `json:"member_count"`     // 成员数量
		MaxMemberCount uint   `json:"max_member_count"` // 最大成员数量
	}

	if err := pa.callAction(context.Background(), "get_group_info", params, &resp); err != nil {
		return nil, err
	}

	return &GroupInfo{
		GroupID:        FormatDiceIDQQGroup(strconv.FormatInt(resp.GroupID, 10)),
		GroupName:      resp.GroupName,
		MemberCount:    resp.MemberCount,
		MaxMemberCount: uint(resp.MaxMemberCount),
	}, nil
}

// FriendDelete removes a friend by ID.
func (pa *PlatformAdapterOB11) FriendDelete(request *FriendOperationRequest) (bool, error) {
	uid, err := parseInt64(ExtractQQUserID(formatAnyID(request.UserID)))
	if err != nil {
		return false, err
	}

	params := map[string]any{"user_id": uid}
	if err := pa.callAction(context.Background(), "delete_friend", params, nil); err != nil {
		return false, err
	}
	return true, nil
}

// FriendAdd is not supported in OneBot 11.
func (pa *PlatformAdapterOB11) FriendAdd(_ *FriendOperationRequest) (bool, error) {
	return false, errors.New("ob11 adapter: friend add not supported")
}

func (pa *PlatformAdapterOB11) convertEventToMessage(evt *ob11EventEnvelope) *types.Message {
	segments := pa.extractSegments(evt.Message)
	if len(segments) == 0 && evt.RawMessage != "" {
		segments = append(segments, &types.TextElement{Content: evt.RawMessage})
	}

	senderID := FormatDiceIDQQ(evt.senderID())
	nickname := evt.senderNickname()
	if nickname == "" {
		nickname = senderID
	}

	msg := &types.Message{
		Platform:    "QQ",
		Time:        evt.Time,
		MessageType: evt.MessageType,
		Segments:    segments,
		Message:     segments.ToText(),
		RawID:       evt.messageID(),
		Sender: types.SenderBase{
			UserID:    senderID,
			Nickname:  nickname,
			GroupRole: evt.senderRole(),
		},
	}

	if evt.MessageType == "group" {
		msg.GroupID = FormatDiceIDQQGroup(evt.groupID())
	}

	if evt.MessageType == "guild" {
		msg.GuildID = evt.guildID()
		msg.ChannelID = evt.channelID()
	}

	if len(msg.Segments) == 0 {
		return nil
	}

	return msg
}

func (pa *PlatformAdapterOB11) extractSegments(raw json.RawMessage) types.MessageSegments {
	var arrayPayload []ob11Segment
	if err := json.Unmarshal(raw, &arrayPayload); err == nil {
		return pa.fromSegments(arrayPayload)
	}

	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil && plain != "" {
		return types.MessageSegments{&types.TextElement{Content: plain}}
	}

	return nil
}

func (pa *PlatformAdapterOB11) fromSegments(src []ob11Segment) types.MessageSegments {
	var result types.MessageSegments
	for _, seg := range src {
		if elem := ob11SegmentToElement(seg); elem != nil {
			result = append(result, elem)
		}
	}
	return result
}

type ob11SendConfig struct {
	messageAction string
	pokeAction    string
	targetParam   string
	targetID      int64
}

func (pa *PlatformAdapterOB11) sendSegmentsWithPoke(ctx context.Context, cfg ob11SendConfig, segments []types.IMessageElement) ([]types.IMessageElement, string, error) {
	sentSegments := make([]types.IMessageElement, 0, len(segments))
	pending := make([]types.IMessageElement, 0, len(segments))
	var lastRawID string

	flushPending := func() error {
		if len(pending) == 0 {
			return nil
		}
		payload := pa.buildMessage(pending)
		pendingSegments := pending
		pending = nil
		if len(payload) == 0 {
			return nil
		}
		params := map[string]any{cfg.targetParam: cfg.targetID, "message": payload}
		var resp ob11SendResponse
		if err := pa.callAction(ctx, cfg.messageAction, params, &resp); err != nil {
			return err
		}
		lastRawID = sanitizeRawMessage(resp.MessageID)
		sentSegments = append(sentSegments, pendingSegments...)
		return nil
	}

	for _, elem := range segments {
		poke, ok := elem.(*types.PokeElement)
		if !ok {
			pending = append(pending, elem)
			continue
		}
		if err := flushPending(); err != nil {
			return sentSegments, lastRawID, err
		}
		if strings.TrimSpace(poke.Target) == "" {
			continue
		}
		pokeTarget, err := parseInt64(poke.Target)
		if err != nil {
			return sentSegments, lastRawID, err
		}
		params := map[string]any{"user_id": pokeTarget}
		if cfg.pokeAction == "group_poke" {
			params[cfg.targetParam] = cfg.targetID
		}
		if err := pa.callAction(ctx, cfg.pokeAction, params, nil); err != nil {
			return sentSegments, lastRawID, err
		}
		sentSegments = append(sentSegments, &types.PokeElement{Target: poke.Target})
	}
	if err := flushPending(); err != nil {
		return sentSegments, lastRawID, err
	}

	if len(sentSegments) == 0 && len(segments) > 0 {
		return nil, "", errors.New("ob11 adapter: no supported segments to send")
	}
	return sentSegments, lastRawID, nil
}

func (pa *PlatformAdapterOB11) buildMessage(segments []types.IMessageElement) []map[string]any {
	log := zap.S().Named("adapter")
	result := make([]map[string]any, 0, len(segments))
	for _, elem := range segments {
		switch e := elem.(type) {
		case *types.TextElement:
			result = append(result, map[string]any{
				"type": "text",
				"data": map[string]any{"text": e.Content},
			})
		case *types.ImageElement:
			file := e.URL
			if file == "" && e.File != nil {
				if e.File.URL != "" {
					file = e.File.URL
				} else {
					file = e.File.File
				}
			}
			if file == "" {
				log.Debug("ob11: skip image with empty source")
				continue
			}
			result = append(result, map[string]any{
				"type": "image",
				"data": map[string]any{"file": file},
			})
		case *types.AtElement:
			result = append(result, map[string]any{
				"type": "at",
				"data": map[string]any{"qq": e.Target},
			})
		case *types.ReplyElement:
			result = append(result, map[string]any{
				"type": "reply",
				"data": map[string]any{"id": e.ReplySeq},
			})
		case *types.RecordElement:
			var file string
			if e.File != nil {
				file = e.File.URL
				if file == "" {
					file = e.File.File
				}
			}
			if file != "" {
				result = append(result, map[string]any{
					"type": "record",
					"data": map[string]any{"file": file},
				})
			}
		case *types.FaceElement:
			result = append(result, map[string]any{
				"type": "face",
				"data": map[string]any{"id": e.FaceID},
			})
		case *types.PokeElement:
			log.Debugf("ob11: poke segment is sent via dedicated action, skip inline serialization target=%s", e.Target)
		default:
			log.Debugf("ob11: unsupported segment type %T", elem)
		}
	}
	return result
}

func (pa *PlatformAdapterOB11) emitEcho(segs []types.IMessageElement, sender *SimpleUserInfo, messageType, targetID, rawID string) {
	if pa.callback == nil {
		return
	}

	segments := types.MessageSegments(segs)

	msg := &types.Message{
		Platform:    "QQ",
		MessageType: messageType,
		Segments:    segments,
		RawID:       rawID,
		Time:        time.Now().Unix(),
		Message:     segments.ToText(),
	}

	if sender != nil {
		msg.Sender.UserID = sender.UserId
		msg.Sender.Nickname = sender.UserName
	}

	switch messageType {
	case "group":
		msg.GroupID = targetID
	case "private":
		msg.Sender.UserID = targetID
		if msg.Sender.Nickname == "" {
			msg.Sender.Nickname = targetID
		}
	}

	pa.callback.OnMessageReceived(&MessageSendCallbackInfo{
		Sender:  sender,
		Message: msg,
	})
}

func formatAnyID(id any) string {
	return stringFromAny(id)
}

func parseInt64(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("ob11 adapter: empty numeric id")
	}
	return strconv.ParseInt(trimmed, 10, 64)
}

func sanitizeRawMessage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(raw))
	s = strings.Trim(s, "\"")
	return s
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	if v, ok := value.(string); ok {
		return v
	}
	if v, ok := value.(json.Number); ok {
		return v.String()
	}
	return fmt.Sprint(value)
}

func segmentDataString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			if str := strings.TrimSpace(stringFromAny(value)); str != "" {
				return str
			}
		}
	}
	return ""
}

func segmentDataStringMap(data map[string]any) map[string]string {
	if len(data) == 0 {
		return nil
	}
	result := make(map[string]string, len(data))
	for key, value := range data {
		result[key] = stringFromAny(value)
	}
	return result
}

func chooseOB11MediaSource(data map[string]any) string {
	file := segmentDataString(data, "file")
	urlValue := segmentDataString(data, "url")
	pathValue := segmentDataString(data, "path")
	return chooseOB11MediaSourceValues(file, urlValue, pathValue)
}

func chooseOB11MediaSourceValues(file, urlValue, pathValue string) string {
	if file != "" {
		if strings.Contains(file, "://") || strings.HasPrefix(file, "base64://") || (urlValue == "" && pathValue == "") {
			return file
		}
		if urlValue != "" {
			return urlValue
		}
		if pathValue != "" {
			return pathValue
		}
	}
	if urlValue != "" {
		return urlValue
	}
	return pathValue
}

func ob11SegmentToElement(seg ob11Segment) types.IMessageElement {
	data := segmentDataStringMap(seg.Data)
	if len(data) == 0 {
		return nil
	}
	if seg.Type == "text" {
		if text := data["text"]; text != "" {
			return &types.TextElement{Content: text}
		}
		return nil
	}
	if seg.Type == "poke" {
		data["qq"] = firstNonEmpty(data["qq"], data["id"], data["user_id"])
	}
	if seg.Type == "image" || seg.Type == "record" || seg.Type == "file" {
		if source := chooseOB11MediaSourceValues(data["file"], data["url"], data["path"]); source != "" {
			data["url"] = source
		}
	}
	elem, _ := types.ElementFromCQData(seg.Type, data)
	return elem
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// getPendingKeys 获取pending map中的所有key，用于调试
func (pa *PlatformAdapterOB11) getPendingKeys() []string {
	return pa.echoWaiter.pendingKeys()
}

func (pa *PlatformAdapterOB11) getDroppedEchoCount() uint64 {
	return pa.echoWaiter.droppedCount()
}

type ob11BaseFrame struct {
	PostType string          `json:"post_type"`
	Echo     json.RawMessage `json:"echo"`
}

type ob11APIResponse struct {
	Status  string          `json:"status"`
	RetCode int64           `json:"retcode"`
	Message string          `json:"message"`
	Wording string          `json:"wording"`
	Data    json.RawMessage `json:"data"`
	Echo    json.RawMessage `json:"echo"`
}

type ob11SendResponse struct {
	MessageID json.RawMessage `json:"message_id"`
}

type ob11Segment struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

type ob11EventEnvelope struct {
	PostType    string          `json:"post_type"`
	MessageType string          `json:"message_type"`
	Time        int64           `json:"time"`
	RawMessage  string          `json:"raw_message"`
	Message     json.RawMessage `json:"message"`
	GroupID     json.RawMessage `json:"group_id"`
	GuildID     json.RawMessage `json:"guild_id"`
	ChannelID   json.RawMessage `json:"channel_id"`
	MessageID   json.RawMessage `json:"message_id"`
	Sender      ob11Sender      `json:"sender"`
}

type ob11NoticeEvent struct {
	PostType    string          `json:"post_type"`
	NoticeType  string          `json:"notice_type"`
	SubType     string          `json:"sub_type"`
	Time        int64           `json:"time"`
	GroupID     json.RawMessage `json:"group_id"`
	GuildID     json.RawMessage `json:"guild_id"`
	ChannelID   json.RawMessage `json:"channel_id"`
	UserID      json.RawMessage `json:"user_id"`
	OperatorID  json.RawMessage `json:"operator_id"`
	TargetID    json.RawMessage `json:"target_id"`
	NoticeEvent json.RawMessage `json:"event"`
}

type ob11RequestEvent struct {
	PostType     string          `json:"post_type"`
	RequestType  string          `json:"request_type"`
	SubType      string          `json:"sub_type"`
	Time         int64           `json:"time"`
	GroupID      json.RawMessage `json:"group_id"`
	GuildID      json.RawMessage `json:"guild_id"`
	ChannelID    json.RawMessage `json:"channel_id"`
	UserID       json.RawMessage `json:"user_id"`
	OperatorID   json.RawMessage `json:"operator_id"`
	Flag         string          `json:"flag"`
	Comment      string          `json:"comment"`
	JoinType     string          `json:"join_type"`
	InvitorID    json.RawMessage `json:"invitor_id"`
	RequestEvent json.RawMessage `json:"event"`
}

type ob11MetaEvent struct {
	PostType      string `json:"post_type"`
	MetaEventType string `json:"meta_event_type"`
	SubType       string `json:"sub_type"`
	Time          int64  `json:"time"`
}

type ob11Sender struct {
	UserID   json.RawMessage `json:"user_id"`
	Nickname string          `json:"nickname"`
	Card     string          `json:"card"`
	Role     string          `json:"role"`
}

func (e *ob11EventEnvelope) senderID() string {
	return sanitizeRawMessage(e.Sender.UserID)
}

func (e *ob11EventEnvelope) senderNickname() string {
	if e.Sender.Card != "" {
		return e.Sender.Card
	}
	return e.Sender.Nickname
}

func (e *ob11EventEnvelope) senderRole() string {
	return e.Sender.Role
}

func (e *ob11EventEnvelope) groupID() string {
	return sanitizeRawMessage(e.GroupID)
}

func (e *ob11EventEnvelope) guildID() string {
	return sanitizeRawMessage(e.GuildID)
}

func (e *ob11EventEnvelope) channelID() string {
	return sanitizeRawMessage(e.ChannelID)
}

func (e *ob11EventEnvelope) messageID() string {
	return sanitizeRawMessage(e.MessageID)
}

// GroupFileList 获取群文件列表
func (pa *PlatformAdapterOB11) GroupFileList(request *GroupFileListRequest) (*GroupFileListResponse, error) {
	params := map[string]any{
		"group_id": ExtractQQGroupID(formatAnyID(request.GroupID)),
	}

	// 根据 Lagrange OneBot 文档，使用不同的 API
	var apiAction string
	if request.FolderID == "" {
		// 获取根目录文件列表
		apiAction = "get_group_root_files"
	} else {
		// 获取指定文件夹的文件列表
		apiAction = "get_group_files_by_folder"
		params["folder_id"] = request.FolderID
	}

	var rawResponse json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()

	err := pa.callAction(ctx, apiAction, params, &rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get group file list: %w", err)
	}

	// 解析响应数据
	var ob11Response struct {
		Files   []map[string]any `json:"files"`
		Folders []map[string]any `json:"folders"`
	}

	if err := json.Unmarshal(rawResponse, &ob11Response); err != nil {
		return nil, fmt.Errorf("failed to parse group file list response: %w", err)
	}

	// 转换为标准格式
	response := &GroupFileListResponse{
		Files:   make([]GroupFileInfo, 0, len(ob11Response.Files)),
		Folders: make([]GroupFolderInfo, 0, len(ob11Response.Folders)),
	}

	// 转换文件信息
	for _, file := range ob11Response.Files {
		fileInfo := GroupFileInfo{}
		if v, ok := file["file_id"].(string); ok {
			fileInfo.FileID = v
		}
		if v, ok := file["file_name"].(string); ok {
			fileInfo.FileName = v
		}
		if v, ok := file["busid"].(float64); ok {
			fileInfo.BusID = int32(v)
		}
		if v, ok := file["file_size"].(float64); ok {
			fileInfo.FileSize = int64(v)
		}
		if v, ok := file["upload_time"].(float64); ok {
			fileInfo.UploadTime = int64(v)
		}
		if v, ok := file["dead_time"].(float64); ok {
			fileInfo.DeadTime = int64(v)
		}
		if v, ok := file["modify_time"].(float64); ok {
			fileInfo.ModifyTime = int64(v)
		}
		if v, ok := file["download_times"].(float64); ok {
			fileInfo.DownloadTimes = int32(v)
		}
		if v, ok := file["uploader"].(float64); ok {
			fileInfo.Uploader = int64(v)
		}
		if v, ok := file["uploader_name"].(string); ok {
			fileInfo.UploaderName = v
		}
		response.Files = append(response.Files, fileInfo)
	}

	// 转换文件夹信息
	for _, folder := range ob11Response.Folders {
		folderInfo := GroupFolderInfo{}
		if v, ok := folder["folder_id"].(string); ok {
			folderInfo.FolderID = v
		}
		if v, ok := folder["folder_name"].(string); ok {
			folderInfo.FolderName = v
		}
		if v, ok := folder["create_time"].(float64); ok {
			folderInfo.CreateTime = int64(v)
		}
		if v, ok := folder["creator"].(float64); ok {
			folderInfo.Creator = int64(v)
		}
		if v, ok := folder["creator_name"].(string); ok {
			folderInfo.CreatorName = v
		}
		if v, ok := folder["total_file_count"].(float64); ok {
			folderInfo.TotalFileCount = int32(v)
		}
		response.Folders = append(response.Folders, folderInfo)
	}

	return response, nil
}

// GroupFileDownload 获取群文件下载链接
func (pa *PlatformAdapterOB11) GroupFileDownload(request *GroupFileDownloadRequest) (*GroupFileDownloadResponse, error) {
	params := map[string]any{
		"group_id": ExtractQQGroupID(formatAnyID(request.GroupID)),
		"file_id":  request.FileID,
	}

	// busid参数在Lagrange OneBot中已废弃，但为了兼容性仍然包含
	if request.BusID != 0 {
		params["busid"] = request.BusID
	}

	var rawResponse json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()

	err := pa.callAction(ctx, "get_group_file_url", params, &rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get group file download url: %w", err)
	}

	// 解析响应数据
	var ob11Response struct {
		URL string `json:"url"`
	}

	if err := json.Unmarshal(rawResponse, &ob11Response); err != nil {
		return nil, fmt.Errorf("failed to parse group file download response: %w", err)
	}

	return &GroupFileDownloadResponse{
		URL: ob11Response.URL,
	}, nil
}

// GroupFileUpload 上传文件到群
func (pa *PlatformAdapterOB11) GroupFileUpload(request *GroupFileUploadRequest) (bool, error) {
	params := map[string]any{
		"group_id": ExtractQQGroupID(formatAnyID(request.GroupID)),
		"file":     request.FilePath, // 仅支持本地文件路径
		"name":     request.FileName, // 文件名称为必需参数
	}

	// 如果指定了文件夹ID，添加到参数中，默认为根目录
	if request.FolderID != "" {
		params["folder"] = request.FolderID
	} else {
		params["folder"] = "/" // 默认上传到根目录
	}

	var rawResponse json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()

	err := pa.callAction(ctx, "upload_group_file", params, &rawResponse)
	if err != nil {
		return false, fmt.Errorf("failed to upload group file: %w", err)
	}

	return true, nil
}
