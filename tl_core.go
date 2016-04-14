package mtproto

import (
	//	"fmt"
	"bytes"
	"compress/gzip"
	"math/big"
)

const (
	layer = 45

	// https://core.telegram.org/schema/mtproto
	crc_vector                     = 0x1cb5c415
	crc_resPQ                      = 0x05162463
	crc_p_q_inner_data             = 0x83c95aec
	crc_server_DH_params_fail      = 0x79cb045d
	crc_server_DH_params_ok        = 0xd0e8075c
	crc_server_DH_inner_data       = 0xb5890dba
	crc_client_DH_inner_data       = 0x6643b654
	crc_dh_gen_ok                  = 0x3bcbf734
	crc_dh_gen_retry               = 0x46dc1fb9
	crc_dh_gen_fail                = 0xa69dae02
	crc_rpc_result                 = 0xf35c6d01
	crc_rpc_error                  = 0x2144ca19
	crc_rpc_answer_unknown         = 0x5e2ad36e
	crc_rpc_answer_dropped_running = 0xcd78e586
	crc_rpc_answer_dropped         = 0xa43ad8b7
	crc_future_salt                = 0x0949d9dc
	crc_future_salts               = 0xae500895
	crc_pong                       = 0x347773c5
	crc_destroy_session_ok         = 0xe22045fc
	crc_destroy_session_none       = 0x62d350c9
	crc_new_session_created        = 0x9ec20908
	crc_msg_container              = 0x73f1f8dc
	//	crc_message                    = 0x5bb8e511
	crc_msg_copy              = 0xe06046b2
	crc_gzip_packed           = 0x3072cfa1
	crc_msgs_ack              = 0x62d6b459
	crc_bad_msg_notification  = 0xa7eff811
	crc_bad_server_salt       = 0xedab447b
	crc_msg_resend_req        = 0x7d861a08
	crc_msgs_state_req        = 0xda69fb52
	crc_msgs_state_info       = 0x04deb57d
	crc_msgs_all_info         = 0x8cc0d131
	crc_msg_detailed_info     = 0x276d3ec6
	crc_msg_new_detailed_info = 0x809db6df
	crc_req_pq                = 0x60469778
	crc_req_DH_params         = 0xd712e4be
	crc_set_client_DH_params  = 0xf5045f1f
	crc_rpc_drop_answer       = 0x58e4a740
	crc_get_future_salts      = 0xb921bd04
	crc_ping                  = 0x7abe77ec
	crc_ping_delay_disconnect = 0xf3427b8c
	crc_destroy_session       = 0xe7512126
	crc_http_wait             = 0x9299359f
)

type TL interface {
	encode() []byte
}

type TL_resPQ struct {
	nonce                          []byte
	server_nonce                   []byte
	pq                             *big.Int
	server_public_key_fingerprints []int64
}

type TL_p_q_inner_data struct {
	pq           *big.Int
	p            *big.Int
	q            *big.Int
	nonce        []byte
	server_nonce []byte
	new_nonce    []byte
}

type TL_server_DH_params_fail struct {
	nonce          []byte
	server_nonce   []byte
	new_nonce_hash []byte
}

type TL_server_DH_params_ok struct {
	nonce            []byte
	server_nonce     []byte
	encrypted_answer []byte
}

type TL_server_DH_inner_data struct {
	nonce        []byte
	server_nonce []byte
	g            int32
	dh_prime     *big.Int
	g_a          *big.Int
	server_time  int32
}

type TL_client_DH_inner_data struct {
	nonce        []byte
	server_nonce []byte
	retry_id     int64
	g_b          *big.Int
}

type TL_dh_gen_ok struct {
	nonce           []byte
	server_nonce    []byte
	new_nonce_hash1 []byte
}

type TL_dh_gen_retry struct {
	nonce           []byte
	server_nonce    []byte
	new_nonce_hash2 []byte
}

type TL_dh_gen_fail struct {
	nonce           []byte
	server_nonce    []byte
	new_nonce_hash3 []byte
}

type TL_rpc_result struct {
	req_msg_id int64
	result     TL
}

type TL_rpc_error struct {
	error_code    int32
	error_message string
}

type TL_rpc_answer_unknown struct {
}

type TL_rpc_answer_dropped_running struct {
}

type TL_rpc_answer_dropped struct {
	msg_id int64
	seq_no int32
	bytes  int32
}

type TL_future_salt struct {
	valid_since int32
	valid_until int32
	salt        []byte
}

type TL_future_salts struct {
	req_msg_id int64
	now        int32
	salts      []TL_future_salt
}

type TL_pong struct {
	msg_id  int64
	ping_id int64
}

type TL_destroy_session_ok struct {
	session_id int64
}

type TL_destroy_session_none struct {
	session_id int64
}

type TL_new_session_created struct {
	first_msg_id int64
	unique_id    int64
	server_salt  []byte
}

type TL_msg_container struct {
	messages []TL_MT_message
}

type TL_MT_message struct {
	msg_id int64
	seqno  int32
	bytes  int32
	body   TL
}

type TL_msg_copy struct {
	orig_message TL_MT_message
}

type TL_gzip_packed struct {
	packed_data []byte
}

type TL_msgs_ack struct {
	msg_ids []int64
}

type TL_bad_msg_notification struct {
	bad_msg_id    int64
	bad_msg_seqno int32
	error_code    int32
}

type TL_bad_server_salt struct {
	bad_msg_id      int64
	bad_msg_seqno   int32
	error_code      int32
	new_server_salt []byte
}

type TL_msg_resend_req struct {
	msg_ids []int64
}

type TL_msgs_state_req struct {
	msg_ids []int64
}

type TL_msgs_state_info struct {
	req_msg_id int64
	info       []byte
}

type TL_msgs_all_info struct {
	msg_ids []int64
	info    []byte
}

type TL_msg_detailed_info struct {
	msg_id        int64
	answer_msg_id int64
	bytes         int32
	status        int32
}

type TL_msg_new_detailed_info struct {
	answer_msg_id int64
	bytes         int32
	status        int32
}

type TL_req_pq struct {
	nonce []byte
}

type TL_req_DH_params struct {
	nonce                  []byte
	server_nonce           []byte
	p                      *big.Int
	q                      *big.Int
	public_key_fingerprint uint64
	encrypted_data         []byte
}

type TL_set_client_DH_params struct {
	nonce          []byte
	server_nonce   []byte
	encrypted_data []byte
}

type TL_rpc_drop_answer struct {
	req_msg_id int64
}

type TL_get_future_salts struct {
	num int32
}

type TL_ping struct {
	ping_id int64
}

type TL_ping_delay_disconnect struct {
	ping_id          int64
	disconnect_delay int32
}

type TL_destroy_session struct {
	session_id int64
}

type TL_http_wait struct {
	max_delay  int32
	wait_after int32
	max_wait   int32
}

// Encode
func (e TL_req_pq) encode() []byte {
	x := NewEncodeBuf(20)
	x.UInt(crc_req_pq)
	x.Bytes(e.nonce)
	return x.buf
}

func (e TL_req_DH_params) encode() []byte {
	x := NewEncodeBuf(512)
	x.UInt(crc_req_DH_params)
	x.Bytes(e.nonce)
	x.Bytes(e.server_nonce)
	x.BigInt(e.p)
	x.BigInt(e.q)
	x.Long(int64(e.public_key_fingerprint))
	x.StringBytes(e.encrypted_data)
	return x.buf
}

func (e TL_p_q_inner_data) encode() []byte {
	x := NewEncodeBuf(256)
	x.UInt(crc_p_q_inner_data)
	x.BigInt(e.pq)
	x.BigInt(e.p)
	x.BigInt(e.q)
	x.Bytes(e.nonce)
	x.Bytes(e.server_nonce)
	x.Bytes(e.new_nonce)
	return x.buf
}

func (e TL_set_client_DH_params) encode() []byte {
	x := NewEncodeBuf(256)
	x.UInt(crc_set_client_DH_params)
	x.Bytes(e.nonce)
	x.Bytes(e.server_nonce)
	x.StringBytes(e.encrypted_data)
	return x.buf
}

func (e TL_rpc_drop_answer) encode() []byte {
	x := NewEncodeBuf(32)
	x.UInt(crc_rpc_drop_answer)
	x.Long(e.req_msg_id)
	return x.buf
}

func (e TL_get_future_salts) encode() []byte {
	x := NewEncodeBuf(32)
	x.UInt(crc_get_future_salts)
	x.Int(e.num)
	return x.buf
}

func (e TL_ping) encode() []byte {
	x := NewEncodeBuf(32)
	x.UInt(crc_ping)
	x.Long(e.ping_id)
	return x.buf
}

func (e TL_ping_delay_disconnect) encode() []byte {
	x := NewEncodeBuf(32)
	x.UInt(crc_ping_delay_disconnect)
	x.Long(e.ping_id)
	x.Int(e.disconnect_delay)
	return x.buf
}

func (e TL_destroy_session) encode() []byte {
	x := NewEncodeBuf(32)
	x.UInt(crc_destroy_session)
	x.Long(e.session_id)
	return x.buf
}

func (e TL_http_wait) encode() []byte {
	x := NewEncodeBuf(32)
	x.UInt(crc_http_wait)
	x.Int(e.max_delay)
	x.Int(e.wait_after)
	x.Int(e.max_wait)
	return x.buf
}

func (e TL_client_DH_inner_data) encode() []byte {
	x := NewEncodeBuf(512)
	x.UInt(crc_client_DH_inner_data)
	x.Bytes(e.nonce)
	x.Bytes(e.server_nonce)
	x.Long(e.retry_id)
	x.BigInt(e.g_b)
	return x.buf
}

func (e TL_pong) encode() []byte {
	x := NewEncodeBuf(32)
	x.UInt(crc_pong)
	x.Long(e.msg_id)
	x.Long(e.ping_id)
	return x.buf
}

func (e TL_msgs_ack) encode() []byte {
	x := NewEncodeBuf(64)
	x.UInt(crc_msgs_ack)
	x.VectorLong(e.msg_ids)
	return x.buf
}

// Easier to just leave these as nil as they will never be needed to be encoded
func (e TL_resPQ) encode() []byte                      { return nil }
func (e TL_server_DH_params_fail) encode() []byte      { return nil }
func (e TL_server_DH_params_ok) encode() []byte        { return nil }
func (e TL_server_DH_inner_data) encode() []byte       { return nil }
func (e TL_dh_gen_ok) encode() []byte                  { return nil }
func (e TL_dh_gen_retry) encode() []byte               { return nil }
func (e TL_dh_gen_fail) encode() []byte                { return nil }
func (e TL_rpc_result) encode() []byte                 { return nil }
func (e TL_rpc_error) encode() []byte                  { return nil }
func (e TL_rpc_answer_unknown) encode() []byte         { return nil }
func (e TL_rpc_answer_dropped_running) encode() []byte { return nil }
func (e TL_rpc_answer_dropped) encode() []byte         { return nil }
func (e TL_future_salt) encode() []byte                { return nil }
func (e TL_future_salts) encode() []byte               { return nil }
func (e TL_destroy_session_ok) encode() []byte         { return nil }
func (e TL_destroy_session_none) encode() []byte       { return nil }
func (e TL_new_session_created) encode() []byte        { return nil }
func (e TL_msg_container) encode() []byte              { return nil }
func (e TL_MT_message) encode() []byte                 { return nil }
func (e TL_msg_copy) encode() []byte                   { return nil }
func (e TL_gzip_packed) encode() []byte                { return nil }
func (e TL_bad_msg_notification) encode() []byte       { return nil }
func (e TL_bad_server_salt) encode() []byte            { return nil }
func (e TL_msg_resend_req) encode() []byte             { return nil }
func (e TL_msgs_state_req) encode() []byte             { return nil }
func (e TL_msgs_state_info) encode() []byte            { return nil }
func (e TL_msgs_all_info) encode() []byte              { return nil }
func (e TL_msg_detailed_info) encode() []byte          { return nil }
func (e TL_msg_new_detailed_info) encode() []byte      { return nil }

// Decode
func (m *DecodeBuf) Object() (r TL) {
	constructor := m.UInt()
	if m.err != nil {
		return nil
	}

	//	fmt.Printf("[%08x]\n", constructor)
	// m.dump()

	switch constructor {
	case crc_resPQ:
		r = TL_resPQ{
			m.Bytes(16),
			m.Bytes(16),
			m.BigInt(),
			m.VectorLong(),
		}

	case crc_p_q_inner_data:
		r = TL_p_q_inner_data{
			m.BigInt(),
			m.BigInt(),
			m.BigInt(),
			m.Bytes(16),
			m.Bytes(16),
			m.Bytes(32),
		}

	case crc_server_DH_params_fail:
		r = TL_server_DH_params_fail{
			m.Bytes(16),
			m.Bytes(16),
			m.Bytes(16),
		}

	case crc_server_DH_params_ok:
		r = TL_server_DH_params_ok{
			m.Bytes(16),
			m.Bytes(16),
			m.StringBytes(),
		}

	case crc_server_DH_inner_data:
		r = TL_server_DH_inner_data{
			m.Bytes(16),
			m.Bytes(16),
			m.Int(),
			m.BigInt(),
			m.BigInt(),
			m.Int(),
		}

	case crc_client_DH_inner_data:
		r = TL_client_DH_inner_data{
			m.Bytes(16),
			m.Bytes(16),
			m.Long(),
			m.BigInt(),
		}

	case crc_dh_gen_ok:
		r = TL_dh_gen_ok{
			m.Bytes(16),
			m.Bytes(16),
			m.Bytes(16),
		}

	case crc_dh_gen_retry:
		r = TL_dh_gen_retry{
			m.Bytes(16),
			m.Bytes(16),
			m.Bytes(16),
		}

	case crc_dh_gen_fail:
		r = TL_dh_gen_fail{
			m.Bytes(16),
			m.Bytes(16),
			m.Bytes(16),
		}

	case crc_rpc_result:
		r = TL_rpc_result{
			m.Long(),
			m.Object(),
		}

	case crc_rpc_error:
		r = TL_rpc_error{
			m.Int(),
			m.String(),
		}

	case crc_rpc_answer_unknown:
		r = TL_rpc_answer_unknown{}

	case crc_rpc_answer_dropped_running:
		r = TL_rpc_answer_dropped_running{}

	case crc_rpc_answer_dropped:
		r = TL_rpc_answer_dropped{
			m.Long(),
			m.Int(),
			m.Int(),
		}

	case crc_future_salt:
		r = TL_future_salt{
			m.Int(),
			m.Int(),
			m.Bytes(8),
		}

	case crc_future_salts:
		r = TL_future_salts{
			m.Long(),
			m.Int(),
			m.Vector_future_salt(),
		}

	case crc_pong:
		r = TL_pong{
			m.Long(),
			m.Long(),
		}

	case crc_destroy_session_ok:
		r = TL_destroy_session_ok{
			m.Long(),
		}

	case crc_destroy_session_none:
		r = TL_destroy_session_none{
			m.Long(),
		}

	case crc_new_session_created:
		r = TL_new_session_created{
			m.Long(),
			m.Long(),
			m.Bytes(8),
		}

	case crc_msg_container:
		size := m.Int()
		arr := make([]TL_MT_message, size)
		for i := int32(0); i < size; i++ {
			arr[i] = TL_MT_message{m.Long(), m.Int(), m.Int(), m.Object()}
			if m.err != nil {
				return nil
			}
		}
		r = TL_msg_container{arr}
		//		r = TL_msg_container{
		//			m.Vector_MT_message(),
		//		}

		//	case crc_message:
		//		r = TL_MT_message{
		//			m.Long(),
		//			m.Int(),
		//			m.Int(),
		//			m.Object(),
		//		}

	case crc_msg_copy:
		r = TL_msg_copy{
			m.Object().(TL_MT_message),
		}

	case crc_gzip_packed:
		obj := make([]byte, 0, 1024)

		var buf bytes.Buffer
		_, _ = buf.Write(m.StringBytes())
		gz, _ := gzip.NewReader(&buf)

		b := make([]byte, 1024)
		for true {
			n, _ := gz.Read(b)
			obj = append(obj, b...)
			if n <= 0 {
				break
			}
		}
		d := NewDecodeBuf(obj)
		r = d.Object()

	case crc_msgs_ack:
		r = TL_msgs_ack{
			m.VectorLong(),
		}

	case crc_bad_msg_notification:
		r = TL_bad_msg_notification{
			m.Long(),
			m.Int(),
			m.Int(),
		}

	case crc_bad_server_salt:
		r = TL_bad_server_salt{
			m.Long(),
			m.Int(),
			m.Int(),
			m.Bytes(8),
		}

	case crc_msg_resend_req:
		r = TL_msg_resend_req{
			m.VectorLong(),
		}

	case crc_msgs_state_req:
		r = TL_msgs_state_req{
			m.VectorLong(),
		}

	case crc_msgs_state_info:
		r = TL_msgs_state_info{
			m.Long(),
			m.StringBytes(),
		}

	case crc_msgs_all_info:
		r = TL_msgs_all_info{
			m.VectorLong(),
			m.StringBytes(),
		}

	case crc_msg_detailed_info:
		r = TL_msg_detailed_info{
			m.Long(),
			m.Long(),
			m.Int(),
			m.Int(),
		}

	case crc_msg_new_detailed_info:
		r = TL_msg_new_detailed_info{
			m.Long(),
			m.Int(),
			m.Int(),
		}

	case crc_req_pq:
		r = TL_req_pq{
			m.Bytes(16),
		}

	case crc_req_DH_params:
		r = TL_req_DH_params{
			m.Bytes(16),
			m.Bytes(16),
			m.BigInt(),
			m.BigInt(),
			uint64(m.Long()),
			m.StringBytes(),
		}

	case crc_set_client_DH_params:
		r = TL_set_client_DH_params{
			m.Bytes(16),
			m.Bytes(16),
			m.StringBytes(),
		}

	case crc_rpc_drop_answer:
		r = TL_rpc_drop_answer{
			m.Long(),
		}

	case crc_get_future_salts:
		r = TL_get_future_salts{
			m.Int(),
		}

	case crc_ping:
		r = TL_ping{
			m.Long(),
		}

	case crc_ping_delay_disconnect:
		r = TL_ping_delay_disconnect{
			m.Long(),
			m.Int(),
		}

	case crc_destroy_session:
		r = TL_destroy_session{
			m.Long(),
		}

	case crc_http_wait:
		r = TL_http_wait{
			m.Int(),
			m.Int(),
			m.Int(),
		}

	default:
		r = m.ObjectGenerated(constructor)

	}

	if m.err != nil {
		return nil
	}

	return
}
