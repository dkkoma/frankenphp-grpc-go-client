//go:build frankengrpc_extension

package frankengrpc

/*
#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif
#include <stdlib.h>
#include "bridge/frankengrpc.h"
*/
import "C"

import (
	"context"
	"errors"
	"sync"
	"unsafe"

	"github.com/dkkoma/frankenphp-grpc-go-client/internal/client"
	"github.com/dkkoma/frankenphp-grpc-go-client/internal/registry"
	"github.com/dunglas/frankenphp"
)

func init() {
	frankenphp.RegisterExtension(unsafe.Pointer(&C.frankengrpc_module_entry))
}

type unaryCall struct {
	mu      sync.Mutex
	channel uint64
	method  string
	cancel  context.CancelFunc
	peer    string
}

type serverStreamingCall struct {
	channel uint64
	method  string
	stream  *client.ServerStream
}

//export fg_channel_new
func fg_channel_new(target *C.char, targetLen C.size_t, options C.fg_channel_options) C.uint64_t {
	ch, err := client.AcquireChannel(goString(target, targetLen), goDialConfig(options))
	if err != nil {
		return 0
	}
	return C.uint64_t(registry.Put(ch))
}

//export fg_channel_close
func fg_channel_close(handle C.uint64_t) {
	ch, err := registry.Get[*client.Channel](uint64(handle))
	if err == nil {
		_ = ch.Close()
	}
	registry.Delete(uint64(handle))
}

func goDialConfig(options C.fg_channel_options) client.DialConfig {
	config := client.DialConfig{
		HasCredentialsPlaceholder: options.has_credentials != 0,
		Authority:                 goString(options.authority, options.authority_len),
		TLSServerNameOverride:     goString(options.ssl_target_name_override, options.ssl_target_name_override_len),
		PrimaryUserAgent:          goString(options.primary_user_agent, options.primary_user_agent_len),
	}
	if options.has_max_receive_message_length != 0 {
		value := int(options.max_receive_message_length)
		config.MaxReceiveMessageLength = &value
	}
	if options.has_max_metadata_size != 0 {
		value := int(options.max_metadata_size)
		config.MaxMetadataSize = &value
	}
	if options.has_absolute_max_metadata_size != 0 {
		value := int(options.absolute_max_metadata_size)
		config.AbsoluteMaxMetadataSize = &value
	}
	return config
}

//export fg_unary_call_new
func fg_unary_call_new(channel C.uint64_t, method *C.char, methodLen C.size_t) C.uint64_t {
	if _, err := registry.Get[*client.Channel](uint64(channel)); err != nil {
		return 0
	}
	return C.uint64_t(registry.Put(&unaryCall{
		channel: uint64(channel),
		method:  goString(method, methodLen),
	}))
}

//export fg_unary_call_free
func fg_unary_call_free(handle C.uint64_t) {
	registry.Delete(uint64(handle))
}

//export fg_unary_start
func fg_unary_start(handle C.uint64_t, payload *C.char, payloadLen C.size_t, md C.fg_metadata, hasTimeout C.int, timeout C.double) C.fg_unary_result {
	call, err := registry.Get[*unaryCall](uint64(handle))
	if err != nil {
		return unaryError(err)
	}
	ch, err := registry.Get[*client.Channel](call.channel)
	if err != nil {
		return unaryError(err)
	}

	req := client.UnaryRequest{
		Method:   call.method,
		Payload:  goBytes(payload, payloadLen),
		Metadata: goMetadata(md),
	}
	if hasTimeout != 0 {
		value := float64(timeout)
		req.TimeoutSeconds = &value
	}

	ctx, cancel := context.WithCancel(context.Background())
	call.mu.Lock()
	call.cancel = cancel
	call.mu.Unlock()
	defer func() {
		call.mu.Lock()
		call.cancel = nil
		call.mu.Unlock()
		cancel()
	}()

	result, err := client.Unary(ctx, ch, req)
	if err != nil {
		return unaryError(err)
	}
	call.peer = result.Peer

	return C.fg_unary_result{
		payload:           cString(result.Payload),
		status:            cStatus(result.Status, result.TrailingMetadata),
		initial_metadata:  cMetadata(result.InitialMetadata),
		trailing_metadata: cMetadata(result.TrailingMetadata),
		peer:              cCharBytes([]byte(result.Peer)),
		peer_len:          C.size_t(len(result.Peer)),
	}
}

//export fg_unary_cancel
func fg_unary_cancel(handle C.uint64_t) {
	call, err := registry.Get[*unaryCall](uint64(handle))
	if err != nil {
		return
	}
	call.mu.Lock()
	cancel := call.cancel
	call.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

//export fg_unary_get_peer
func fg_unary_get_peer(handle C.uint64_t) C.fg_string {
	call, err := registry.Get[*unaryCall](uint64(handle))
	if err != nil {
		return C.fg_string{}
	}
	return cString([]byte(call.peer))
}

//export fg_server_streaming_call_new
func fg_server_streaming_call_new(channel C.uint64_t, method *C.char, methodLen C.size_t) C.uint64_t {
	if _, err := registry.Get[*client.Channel](uint64(channel)); err != nil {
		return 0
	}
	return C.uint64_t(registry.Put(&serverStreamingCall{
		channel: uint64(channel),
		method:  goString(method, methodLen),
	}))
}

//export fg_server_streaming_call_free
func fg_server_streaming_call_free(handle C.uint64_t) {
	call, err := registry.Get[*serverStreamingCall](uint64(handle))
	if err == nil && call.stream != nil {
		call.stream.Cancel()
	}
	registry.Delete(uint64(handle))
}

//export fg_stream_start
func fg_stream_start(handle C.uint64_t, payload *C.char, payloadLen C.size_t, md C.fg_metadata, hasTimeout C.int, timeout C.double) C.fg_error {
	call, err := registry.Get[*serverStreamingCall](uint64(handle))
	if err != nil {
		return cError(err)
	}
	ch, err := registry.Get[*client.Channel](call.channel)
	if err != nil {
		return cError(err)
	}

	req := client.ServerStreamingRequest{
		Method:   call.method,
		Payload:  goBytes(payload, payloadLen),
		Metadata: goMetadata(md),
	}
	if hasTimeout != 0 {
		value := float64(timeout)
		req.TimeoutSeconds = &value
	}

	stream, err := client.StartServerStream(context.Background(), ch, req)
	if err != nil {
		return cError(err)
	}
	call.stream = stream
	return C.fg_error{}
}

//export fg_stream_read
func fg_stream_read(handle C.uint64_t) C.fg_stream_read_result {
	call, err := registry.Get[*serverStreamingCall](uint64(handle))
	if err != nil {
		return streamReadError(err)
	}
	if call.stream == nil {
		return streamReadError(errors.New("frankengrpc: stream has not been started"))
	}
	payload, ok, err := call.stream.Read()
	if err != nil {
		return streamReadError(err)
	}
	return C.fg_stream_read_result{
		has_message: boolInt(ok),
		payload:     cString(payload),
	}
}

//export fg_stream_get_initial_metadata
func fg_stream_get_initial_metadata(handle C.uint64_t) C.fg_metadata {
	call, err := registry.Get[*serverStreamingCall](uint64(handle))
	if err != nil || call.stream == nil {
		return C.fg_metadata{}
	}
	return cMetadata(call.stream.InitialMetadata())
}

//export fg_stream_get_trailing_metadata
func fg_stream_get_trailing_metadata(handle C.uint64_t) C.fg_metadata {
	call, err := registry.Get[*serverStreamingCall](uint64(handle))
	if err != nil || call.stream == nil {
		return C.fg_metadata{}
	}
	return cMetadata(call.stream.TrailingMetadata())
}

//export fg_stream_get_status
func fg_stream_get_status(handle C.uint64_t) C.fg_status {
	call, err := registry.Get[*serverStreamingCall](uint64(handle))
	if err != nil || call.stream == nil {
		return cStatus(client.Status{Code: 2, Details: "frankengrpc: stream has not been started"}, nil)
	}
	return cStatus(call.stream.Status(), call.stream.TrailingMetadata())
}

//export fg_stream_cancel
func fg_stream_cancel(handle C.uint64_t) {
	call, err := registry.Get[*serverStreamingCall](uint64(handle))
	if err == nil && call.stream != nil {
		call.stream.Cancel()
	}
}

//export fg_stream_get_peer
func fg_stream_get_peer(handle C.uint64_t) C.fg_string {
	call, err := registry.Get[*serverStreamingCall](uint64(handle))
	if err != nil || call.stream == nil {
		return C.fg_string{}
	}
	return cString([]byte(call.stream.Peer()))
}

//export fg_free_string
func fg_free_string(value C.fg_string) {
	C.free(unsafe.Pointer(value.data))
}

//export fg_free_metadata
func fg_free_metadata(md C.fg_metadata) {
	freeMetadata(md)
}

//export fg_free_status
func fg_free_status(status C.fg_status) {
	C.free(unsafe.Pointer(status.details))
	freeMetadata(status.metadata)
}

//export fg_free_error
func fg_free_error(err C.fg_error) {
	C.free(unsafe.Pointer(err.error))
}

//export fg_free_unary_result
func fg_free_unary_result(result C.fg_unary_result) {
	fg_free_error(result.error)
	fg_free_string(result.payload)
	fg_free_status(result.status)
	fg_free_metadata(result.initial_metadata)
	fg_free_metadata(result.trailing_metadata)
	C.free(unsafe.Pointer(result.peer))
}

//export fg_free_stream_read_result
func fg_free_stream_read_result(result C.fg_stream_read_result) {
	fg_free_error(result.error)
	fg_free_string(result.payload)
}

func unaryError(err error) C.fg_unary_result {
	return C.fg_unary_result{error: cError(err)}
}

func streamReadError(err error) C.fg_stream_read_result {
	return C.fg_stream_read_result{error: cError(err)}
}

func cError(err error) C.fg_error {
	message := []byte(err.Error())
	return C.fg_error{
		has_error: boolInt(true),
		error:     cCharBytes(message),
		error_len: C.size_t(len(message)),
	}
}

func boolInt(value bool) C.int {
	if value {
		return 1
	}
	return 0
}

func goString(data *C.char, length C.size_t) string {
	return string(goBytes(data, length))
}

func goBytes(data *C.char, length C.size_t) []byte {
	if data == nil || length == 0 {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(data), C.int(length))
}

func goMetadata(md C.fg_metadata) client.Metadata {
	if md.entries == nil || md.entries_len == 0 {
		return nil
	}

	out := client.Metadata{}
	entries := unsafe.Slice(md.entries, int(md.entries_len))
	for _, entry := range entries {
		key := goString(entry.key, entry.key_len)
		values := unsafe.Slice(entry.values, int(entry.values_len))
		for _, value := range values {
			out[key] = append(out[key], goString(value.data, value.len))
		}
	}
	return out
}

func cStatus(status client.Status, md client.Metadata) C.fg_status {
	details := []byte(status.Details)
	return C.fg_status{
		code:        C.int(status.Code),
		details:     cCharBytes(details),
		details_len: C.size_t(len(details)),
		metadata:    cMetadata(md),
	}
}

func cMetadata(md client.Metadata) C.fg_metadata {
	if len(md) == 0 {
		return C.fg_metadata{}
	}

	entrySize := C.size_t(unsafe.Sizeof(C.fg_metadata_entry{}))
	entriesPtr := C.malloc(C.size_t(len(md)) * entrySize)
	entries := unsafe.Slice((*C.fg_metadata_entry)(entriesPtr), len(md))

	i := 0
	for key, values := range md {
		valueSize := C.size_t(unsafe.Sizeof(C.fg_string{}))
		valuesPtr := C.malloc(C.size_t(len(values)) * valueSize)
		cValues := unsafe.Slice((*C.fg_string)(valuesPtr), len(values))
		for j, value := range values {
			cValues[j] = cString([]byte(value))
		}
		entries[i] = C.fg_metadata_entry{
			values:     (*C.fg_string)(valuesPtr),
			values_len: C.size_t(len(values)),
			key:        cCharBytes([]byte(key)),
			key_len:    C.size_t(len(key)),
		}
		i++
	}

	return C.fg_metadata{
		entries:     (*C.fg_metadata_entry)(entriesPtr),
		entries_len: C.size_t(len(md)),
	}
}

func cString(data []byte) C.fg_string {
	return C.fg_string{
		data: cCharBytes(data),
		len:  C.size_t(len(data)),
	}
}

func cCharBytes(data []byte) *C.char {
	if len(data) == 0 {
		return nil
	}
	return (*C.char)(C.CBytes(data))
}

func freeMetadata(md C.fg_metadata) {
	if md.entries == nil || md.entries_len == 0 {
		return
	}
	entries := unsafe.Slice(md.entries, int(md.entries_len))
	for _, entry := range entries {
		C.free(unsafe.Pointer(entry.key))
		if entry.values != nil {
			values := unsafe.Slice(entry.values, int(entry.values_len))
			for _, value := range values {
				C.free(unsafe.Pointer(value.data))
			}
			C.free(unsafe.Pointer(entry.values))
		}
	}
	C.free(unsafe.Pointer(md.entries))
}
