//go:build frankengrpc_extension

#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif
#include <php.h>
#include <Zend/zend_exceptions.h>
#include <Zend/zend_interfaces.h>
#include "bridge/frankengrpc.h"
#include "_cgo_export.h"

static zend_class_entry *channel_ce;
static zend_class_entry *unary_call_ce;
static zend_class_entry *server_streaming_call_ce;
static zend_class_entry *unary_result_ce;
static zend_class_entry *status_ce;

typedef struct {
    uint64_t handle;
    zend_object std;
} fg_handle_object;

typedef struct {
    zval channel;
    zend_string *method;
    zend_string *peer;
    zend_object std;
} fg_unary_call_object;

static inline fg_handle_object *fg_handle_from_obj(zend_object *obj)
{
    return (fg_handle_object *)((char *)(obj) - XtOffsetOf(fg_handle_object, std));
}

static inline fg_unary_call_object *fg_unary_call_from_obj(zend_object *obj)
{
    return (fg_unary_call_object *)((char *)(obj) - XtOffsetOf(fg_unary_call_object, std));
}

#define FG_HANDLE_P(zv) fg_handle_from_obj(Z_OBJ_P((zv)))
#define FG_UNARY_CALL_P(zv) fg_unary_call_from_obj(Z_OBJ_P((zv)))

static zend_object_handlers channel_handlers;
static zend_object_handlers unary_call_handlers;
static zend_object_handlers server_streaming_call_handlers;

static zend_object *fg_channel_create(zend_class_entry *ce)
{
    fg_handle_object *obj = ecalloc(1, sizeof(fg_handle_object) + zend_object_properties_size(ce));
    zend_object_std_init(&obj->std, ce);
    object_properties_init(&obj->std, ce);
    obj->std.handlers = &channel_handlers;
    return &obj->std;
}

static zend_object *fg_unary_call_create(zend_class_entry *ce)
{
    fg_unary_call_object *obj = ecalloc(1, sizeof(fg_unary_call_object) + zend_object_properties_size(ce));
    zend_object_std_init(&obj->std, ce);
    object_properties_init(&obj->std, ce);
    ZVAL_UNDEF(&obj->channel);
    obj->std.handlers = &unary_call_handlers;
    return &obj->std;
}

static zend_object *fg_server_streaming_call_create(zend_class_entry *ce)
{
    fg_handle_object *obj = ecalloc(1, sizeof(fg_handle_object) + zend_object_properties_size(ce));
    zend_object_std_init(&obj->std, ce);
    object_properties_init(&obj->std, ce);
    obj->std.handlers = &server_streaming_call_handlers;
    return &obj->std;
}

static void fg_channel_free(zend_object *object)
{
    fg_handle_object *obj = fg_handle_from_obj(object);
    if (obj->handle != 0) {
        fg_channel_close(obj->handle);
        obj->handle = 0;
    }
    zend_object_std_dtor(&obj->std);
}

static void fg_unary_call_free_obj(zend_object *object)
{
    fg_unary_call_object *obj = fg_unary_call_from_obj(object);
    if (!Z_ISUNDEF(obj->channel)) {
        zval_ptr_dtor(&obj->channel);
        ZVAL_UNDEF(&obj->channel);
    }
    if (obj->method != NULL) {
        zend_string_release(obj->method);
        obj->method = NULL;
    }
    if (obj->peer != NULL) {
        zend_string_release(obj->peer);
        obj->peer = NULL;
    }
    zend_object_std_dtor(&obj->std);
}

static void fg_server_streaming_call_free_obj(zend_object *object)
{
    fg_handle_object *obj = fg_handle_from_obj(object);
    if (obj->handle != 0) {
        fg_server_streaming_call_free(obj->handle);
        obj->handle = 0;
    }
    zend_object_std_dtor(&obj->std);
}

static void fg_throw_error(fg_error error)
{
    zend_throw_exception(zend_ce_exception, error.error ? error.error : "FrankenGrpc native error", 0);
}

static void fg_metadata_to_zval(fg_metadata md, zval *return_value)
{
    array_init(return_value);
    for (size_t i = 0; i < md.entries_len; i++) {
        fg_metadata_entry entry = md.entries[i];
        zval values;
        array_init(&values);
        for (size_t j = 0; j < entry.values_len; j++) {
            fg_string value = entry.values[j];
            add_next_index_stringl(&values, value.data ? value.data : "", value.len);
        }
        add_assoc_zval_ex(return_value, entry.key, entry.key_len, &values);
    }
}

static void fg_status_to_object(fg_status status, zval *return_value)
{
    zval metadata;
    object_init_ex(return_value, status_ce);
    fg_metadata_to_zval(status.metadata, &metadata);
    zend_update_property_long(status_ce, Z_OBJ_P(return_value), "code", sizeof("code") - 1, status.code);
    zend_update_property_stringl(status_ce, Z_OBJ_P(return_value), "details", sizeof("details") - 1, status.details ? status.details : "", status.details_len);
    zend_update_property(status_ce, Z_OBJ_P(return_value), "metadata", sizeof("metadata") - 1, &metadata);
    zval_ptr_dtor(&metadata);
}

static void fg_unary_result_to_object(fg_unary_result result, zval *return_value)
{
    zval status;
    zval initial_metadata;
    zval trailing_metadata;

    object_init_ex(return_value, unary_result_ce);
    fg_status_to_object(result.status, &status);
    fg_metadata_to_zval(result.initial_metadata, &initial_metadata);
    fg_metadata_to_zval(result.trailing_metadata, &trailing_metadata);

    zend_update_property_stringl(unary_result_ce, Z_OBJ_P(return_value), "payload", sizeof("payload") - 1, result.payload.data ? result.payload.data : "", result.payload.len);
    zend_update_property(unary_result_ce, Z_OBJ_P(return_value), "status", sizeof("status") - 1, &status);
    zend_update_property(unary_result_ce, Z_OBJ_P(return_value), "initialMetadata", sizeof("initialMetadata") - 1, &initial_metadata);
    zend_update_property(unary_result_ce, Z_OBJ_P(return_value), "trailingMetadata", sizeof("trailingMetadata") - 1, &trailing_metadata);

    zval_ptr_dtor(&status);
    zval_ptr_dtor(&initial_metadata);
    zval_ptr_dtor(&trailing_metadata);
}

static bool fg_metadata_from_zval(zval *metadata, fg_metadata *out)
{
    HashTable *table = Z_ARRVAL_P(metadata);
    out->entries_len = zend_hash_num_elements(table);
    out->entries = out->entries_len == 0 ? NULL : ecalloc(out->entries_len, sizeof(fg_metadata_entry));

    size_t i = 0;
    zend_string *key;
    zval *value;
    ZEND_HASH_FOREACH_STR_KEY_VAL(table, key, value) {
        if (key == NULL || Z_TYPE_P(value) != IS_ARRAY) {
            zend_throw_exception(zend_ce_type_error, "metadata must be array<string, list<string>>", 0);
            return false;
        }

        HashTable *values_table = Z_ARRVAL_P(value);
        out->entries[i].key = ZSTR_VAL(key);
        out->entries[i].key_len = ZSTR_LEN(key);
        out->entries[i].values_len = zend_hash_num_elements(values_table);
        out->entries[i].values = out->entries[i].values_len == 0 ? NULL : ecalloc(out->entries[i].values_len, sizeof(fg_string));

        size_t j = 0;
        zval *item;
        ZEND_HASH_FOREACH_VAL(values_table, item) {
            if (Z_TYPE_P(item) != IS_STRING) {
                zend_throw_exception(zend_ce_type_error, "metadata values must be strings", 0);
                return false;
            }
            out->entries[i].values[j].data = Z_STRVAL_P(item);
            out->entries[i].values[j].len = Z_STRLEN_P(item);
            j++;
        } ZEND_HASH_FOREACH_END();
        i++;
    } ZEND_HASH_FOREACH_END();

    return true;
}

static void fg_free_input_metadata(fg_metadata *metadata)
{
    if (metadata->entries == NULL) {
        return;
    }
    for (size_t i = 0; i < metadata->entries_len; i++) {
        if (metadata->entries[i].values != NULL) {
            efree(metadata->entries[i].values);
        }
    }
    efree(metadata->entries);
    metadata->entries = NULL;
    metadata->entries_len = 0;
}

static void fg_channel_option_string(HashTable *table, const char *key, size_t key_len, char **out, size_t *out_len)
{
    zval *value = zend_hash_str_find(table, key, key_len);
    if (value == NULL || Z_TYPE_P(value) != IS_STRING) {
        return;
    }
    *out = Z_STRVAL_P(value);
    *out_len = Z_STRLEN_P(value);
}

static void fg_channel_option_long(HashTable *table, const char *key, size_t key_len, zend_long min, int *has_value, int *out)
{
    zval *value = zend_hash_str_find(table, key, key_len);
    if (value == NULL || Z_TYPE_P(value) != IS_LONG || Z_LVAL_P(value) < min || Z_LVAL_P(value) > INT32_MAX) {
        return;
    }
    *has_value = 1;
    *out = (int) Z_LVAL_P(value);
}

static fg_channel_options fg_channel_options_from_zval(zval *options)
{
    fg_channel_options out = {0};
    if (options == NULL) {
        return out;
    }

    HashTable *table = Z_ARRVAL_P(options);
    zval *credentials = zend_hash_str_find(table, "credentials", sizeof("credentials") - 1);
    if (credentials != NULL) {
        out.has_credentials = 1;
    }

    fg_channel_option_string(table, "grpc.default_authority", sizeof("grpc.default_authority") - 1, &out.authority, &out.authority_len);
    fg_channel_option_string(table, "grpc.ssl_target_name_override", sizeof("grpc.ssl_target_name_override") - 1, &out.ssl_target_name_override, &out.ssl_target_name_override_len);
    fg_channel_option_string(table, "grpc.primary_user_agent", sizeof("grpc.primary_user_agent") - 1, &out.primary_user_agent, &out.primary_user_agent_len);
    fg_channel_option_long(table, "grpc.max_receive_message_length", sizeof("grpc.max_receive_message_length") - 1, -1, &out.has_max_receive_message_length, &out.max_receive_message_length);
    fg_channel_option_long(table, "grpc.max_metadata_size", sizeof("grpc.max_metadata_size") - 1, 0, &out.has_max_metadata_size, &out.max_metadata_size);
    fg_channel_option_long(table, "grpc.absolute_max_metadata_size", sizeof("grpc.absolute_max_metadata_size") - 1, 0, &out.has_absolute_max_metadata_size, &out.absolute_max_metadata_size);

    return out;
}

ZEND_BEGIN_ARG_INFO_EX(arginfo_channel_construct, 0, 0, 1)
    ZEND_ARG_TYPE_INFO(0, target, IS_STRING, 0)
    ZEND_ARG_TYPE_INFO_WITH_DEFAULT_VALUE(0, options, IS_ARRAY, 0, "[]")
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_void, 0, 0, IS_VOID, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_INFO_EX(arginfo_unary_call_construct, 0, 0, 2)
    ZEND_ARG_OBJ_INFO(0, channel, FrankenGrpc\\Channel, 0)
    ZEND_ARG_TYPE_INFO(0, method, IS_STRING, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_OBJ_INFO_EX(arginfo_unary_start, 0, 1, FrankenGrpc\\UnaryResult, 0)
    ZEND_ARG_TYPE_INFO(0, payload, IS_STRING, 0)
    ZEND_ARG_TYPE_INFO_WITH_DEFAULT_VALUE(0, metadata, IS_ARRAY, 0, "[]")
    ZEND_ARG_TYPE_INFO_WITH_DEFAULT_VALUE(0, timeoutSeconds, IS_DOUBLE, 1, "null")
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_get_peer, 0, 0, IS_STRING, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_INFO_EX(arginfo_stream_construct, 0, 0, 2)
    ZEND_ARG_OBJ_INFO(0, channel, FrankenGrpc\\Channel, 0)
    ZEND_ARG_TYPE_INFO(0, method, IS_STRING, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_stream_start, 0, 1, IS_VOID, 0)
    ZEND_ARG_TYPE_INFO(0, payload, IS_STRING, 0)
    ZEND_ARG_TYPE_INFO_WITH_DEFAULT_VALUE(0, metadata, IS_ARRAY, 0, "[]")
    ZEND_ARG_TYPE_INFO_WITH_DEFAULT_VALUE(0, timeoutSeconds, IS_DOUBLE, 1, "null")
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_stream_read, 0, 0, IS_STRING, 1)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_metadata, 0, 0, IS_ARRAY, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_OBJ_INFO_EX(arginfo_status_return, 0, 0, FrankenGrpc\\Status, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_INFO_EX(arginfo_status_construct, 0, 0, 1)
    ZEND_ARG_TYPE_INFO(0, code, IS_LONG, 0)
    ZEND_ARG_TYPE_INFO_WITH_DEFAULT_VALUE(0, details, IS_STRING, 0, "''")
    ZEND_ARG_TYPE_INFO_WITH_DEFAULT_VALUE(0, metadata, IS_ARRAY, 0, "[]")
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_INFO_EX(arginfo_unary_result_construct, 0, 0, 2)
    ZEND_ARG_TYPE_INFO(0, payload, IS_STRING, 0)
    ZEND_ARG_OBJ_INFO(0, status, FrankenGrpc\\Status, 0)
    ZEND_ARG_TYPE_INFO_WITH_DEFAULT_VALUE(0, initialMetadata, IS_ARRAY, 0, "[]")
    ZEND_ARG_TYPE_INFO_WITH_DEFAULT_VALUE(0, trailingMetadata, IS_ARRAY, 0, "[]")
ZEND_END_ARG_INFO()

PHP_METHOD(FrankenGrpc_Channel, __construct)
{
    zend_string *target;
    zval *options = NULL;
    fg_channel_options channel_options = {0};
    ZEND_PARSE_PARAMETERS_START(1, 2)
        Z_PARAM_STR(target)
        Z_PARAM_OPTIONAL
        Z_PARAM_ARRAY(options)
    ZEND_PARSE_PARAMETERS_END();

    channel_options = fg_channel_options_from_zval(options);

    fg_handle_object *obj = FG_HANDLE_P(ZEND_THIS);
    obj->handle = fg_channel_new(ZSTR_VAL(target), ZSTR_LEN(target), channel_options);
    if (obj->handle == 0) {
        zend_throw_exception(zend_ce_exception, "failed to create FrankenGrpc channel", 0);
    }
}

PHP_METHOD(FrankenGrpc_Channel, close)
{
    ZEND_PARSE_PARAMETERS_NONE();
    fg_handle_object *obj = FG_HANDLE_P(ZEND_THIS);
    if (obj->handle != 0) {
        fg_channel_close(obj->handle);
        obj->handle = 0;
    }
}

PHP_METHOD(FrankenGrpc_UnaryCall, __construct)
{
    zval *channel;
    zend_string *method;
    ZEND_PARSE_PARAMETERS_START(2, 2)
        Z_PARAM_OBJECT_OF_CLASS(channel, channel_ce)
        Z_PARAM_STR(method)
    ZEND_PARSE_PARAMETERS_END();

    fg_handle_object *channel_obj = FG_HANDLE_P(channel);
    if (channel_obj->handle == 0) {
        zend_throw_exception(zend_ce_exception, "failed to create FrankenGrpc unary call", 0);
        RETURN_THROWS();
    }

    fg_unary_call_object *obj = FG_UNARY_CALL_P(ZEND_THIS);
    if (!Z_ISUNDEF(obj->channel)) {
        zval_ptr_dtor(&obj->channel);
        ZVAL_UNDEF(&obj->channel);
    }
    if (obj->method != NULL) {
        zend_string_release(obj->method);
    }
    if (obj->peer != NULL) {
        zend_string_release(obj->peer);
        obj->peer = NULL;
    }
    ZVAL_COPY(&obj->channel, channel);
    obj->method = zend_string_copy(method);
}

PHP_METHOD(FrankenGrpc_UnaryCall, start)
{
    zend_string *payload;
    zval *metadata = NULL;
    double timeout = 0;
    bool timeout_is_null = true;
    fg_metadata input_metadata = {0};

    ZEND_PARSE_PARAMETERS_START(1, 3)
        Z_PARAM_STR(payload)
        Z_PARAM_OPTIONAL
        Z_PARAM_ARRAY(metadata)
        Z_PARAM_DOUBLE_OR_NULL(timeout, timeout_is_null)
    ZEND_PARSE_PARAMETERS_END();

    if (metadata == NULL) {
        zval empty;
        array_init(&empty);
        metadata = &empty;
        fg_metadata_from_zval(metadata, &input_metadata);
        zval_ptr_dtor(&empty);
    } else if (!fg_metadata_from_zval(metadata, &input_metadata)) {
        fg_free_input_metadata(&input_metadata);
        RETURN_THROWS();
    }

    fg_unary_call_object *obj = FG_UNARY_CALL_P(ZEND_THIS);
    fg_handle_object *channel_obj = FG_HANDLE_P(&obj->channel);
    fg_unary_result result = fg_unary_start(channel_obj->handle, ZSTR_VAL(obj->method), ZSTR_LEN(obj->method), ZSTR_VAL(payload), ZSTR_LEN(payload), input_metadata, timeout_is_null ? 0 : 1, timeout);
    fg_free_input_metadata(&input_metadata);
    if (result.error.has_error) {
        fg_throw_error(result.error);
        fg_free_unary_result(result);
        RETURN_THROWS();
    }
    if (obj->peer != NULL) {
        zend_string_release(obj->peer);
    }
    obj->peer = zend_string_init(result.peer ? result.peer : "", result.peer_len, 0);
    fg_unary_result_to_object(result, return_value);
    fg_free_unary_result(result);
}

PHP_METHOD(FrankenGrpc_UnaryCall, cancel)
{
    ZEND_PARSE_PARAMETERS_NONE();
}

PHP_METHOD(FrankenGrpc_UnaryCall, getPeer)
{
    ZEND_PARSE_PARAMETERS_NONE();
    fg_unary_call_object *obj = FG_UNARY_CALL_P(ZEND_THIS);
    if (obj->peer == NULL) {
        RETURN_EMPTY_STRING();
    }
    RETURN_STR_COPY(obj->peer);
}

PHP_METHOD(FrankenGrpc_ServerStreamingCall, __construct)
{
    zval *channel;
    zend_string *method;
    ZEND_PARSE_PARAMETERS_START(2, 2)
        Z_PARAM_OBJECT_OF_CLASS(channel, channel_ce)
        Z_PARAM_STR(method)
    ZEND_PARSE_PARAMETERS_END();

    fg_handle_object *channel_obj = FG_HANDLE_P(channel);
    fg_handle_object *obj = FG_HANDLE_P(ZEND_THIS);
    obj->handle = fg_server_streaming_call_new(channel_obj->handle, ZSTR_VAL(method), ZSTR_LEN(method));
    if (obj->handle == 0) {
        zend_throw_exception(zend_ce_exception, "failed to create FrankenGrpc server streaming call", 0);
    }
}

PHP_METHOD(FrankenGrpc_ServerStreamingCall, start)
{
    zend_string *payload;
    zval *metadata = NULL;
    double timeout = 0;
    bool timeout_is_null = true;
    fg_metadata input_metadata = {0};

    ZEND_PARSE_PARAMETERS_START(1, 3)
        Z_PARAM_STR(payload)
        Z_PARAM_OPTIONAL
        Z_PARAM_ARRAY(metadata)
        Z_PARAM_DOUBLE_OR_NULL(timeout, timeout_is_null)
    ZEND_PARSE_PARAMETERS_END();

    if (metadata == NULL) {
        zval empty;
        array_init(&empty);
        metadata = &empty;
        fg_metadata_from_zval(metadata, &input_metadata);
        zval_ptr_dtor(&empty);
    } else if (!fg_metadata_from_zval(metadata, &input_metadata)) {
        fg_free_input_metadata(&input_metadata);
        RETURN_THROWS();
    }

    fg_handle_object *obj = FG_HANDLE_P(ZEND_THIS);
    fg_error error = fg_stream_start(obj->handle, ZSTR_VAL(payload), ZSTR_LEN(payload), input_metadata, timeout_is_null ? 0 : 1, timeout);
    fg_free_input_metadata(&input_metadata);
    if (error.has_error) {
        fg_throw_error(error);
        fg_free_error(error);
        RETURN_THROWS();
    }
}

PHP_METHOD(FrankenGrpc_ServerStreamingCall, read)
{
    ZEND_PARSE_PARAMETERS_NONE();
    fg_handle_object *obj = FG_HANDLE_P(ZEND_THIS);
    fg_stream_read_result result = fg_stream_read(obj->handle);
    if (result.error.has_error) {
        fg_throw_error(result.error);
        fg_free_stream_read_result(result);
        RETURN_THROWS();
    }
    if (!result.has_message) {
        fg_free_stream_read_result(result);
        RETURN_NULL();
    }
    RETVAL_STRINGL(result.payload.data ? result.payload.data : "", result.payload.len);
    fg_free_stream_read_result(result);
}

PHP_METHOD(FrankenGrpc_ServerStreamingCall, getInitialMetadata)
{
    ZEND_PARSE_PARAMETERS_NONE();
    fg_handle_object *obj = FG_HANDLE_P(ZEND_THIS);
    fg_metadata md = fg_stream_get_initial_metadata(obj->handle);
    fg_metadata_to_zval(md, return_value);
    fg_free_metadata(md);
}

PHP_METHOD(FrankenGrpc_ServerStreamingCall, getStatus)
{
    ZEND_PARSE_PARAMETERS_NONE();
    fg_handle_object *obj = FG_HANDLE_P(ZEND_THIS);
    fg_status status = fg_stream_get_status(obj->handle);
    fg_status_to_object(status, return_value);
    fg_free_status(status);
}

PHP_METHOD(FrankenGrpc_ServerStreamingCall, getTrailingMetadata)
{
    ZEND_PARSE_PARAMETERS_NONE();
    fg_handle_object *obj = FG_HANDLE_P(ZEND_THIS);
    fg_metadata md = fg_stream_get_trailing_metadata(obj->handle);
    fg_metadata_to_zval(md, return_value);
    fg_free_metadata(md);
}

PHP_METHOD(FrankenGrpc_ServerStreamingCall, cancel)
{
    ZEND_PARSE_PARAMETERS_NONE();
    fg_handle_object *obj = FG_HANDLE_P(ZEND_THIS);
    fg_stream_cancel(obj->handle);
}

PHP_METHOD(FrankenGrpc_ServerStreamingCall, getPeer)
{
    ZEND_PARSE_PARAMETERS_NONE();
    fg_handle_object *obj = FG_HANDLE_P(ZEND_THIS);
    fg_string peer = fg_stream_get_peer(obj->handle);
    RETVAL_STRINGL(peer.data ? peer.data : "", peer.len);
    fg_free_string(peer);
}

PHP_METHOD(FrankenGrpc_Status, __construct)
{
    zend_long code;
    zend_string *details = NULL;
    zval *metadata = NULL;
    ZEND_PARSE_PARAMETERS_START(1, 3)
        Z_PARAM_LONG(code)
        Z_PARAM_OPTIONAL
        Z_PARAM_STR(details)
        Z_PARAM_ARRAY(metadata)
    ZEND_PARSE_PARAMETERS_END();

    zval empty;
    if (metadata == NULL) {
        array_init(&empty);
        metadata = &empty;
    }
    zend_update_property_long(status_ce, Z_OBJ_P(ZEND_THIS), "code", sizeof("code") - 1, code);
    zend_update_property_stringl(status_ce, Z_OBJ_P(ZEND_THIS), "details", sizeof("details") - 1, details ? ZSTR_VAL(details) : "", details ? ZSTR_LEN(details) : 0);
    zend_update_property(status_ce, Z_OBJ_P(ZEND_THIS), "metadata", sizeof("metadata") - 1, metadata);
    if (metadata == &empty) {
        zval_ptr_dtor(&empty);
    }
}

PHP_METHOD(FrankenGrpc_UnaryResult, __construct)
{
    zend_string *payload;
    zval *status;
    zval *initial_metadata = NULL;
    zval *trailing_metadata = NULL;
    ZEND_PARSE_PARAMETERS_START(2, 4)
        Z_PARAM_STR(payload)
        Z_PARAM_OBJECT_OF_CLASS(status, status_ce)
        Z_PARAM_OPTIONAL
        Z_PARAM_ARRAY(initial_metadata)
        Z_PARAM_ARRAY(trailing_metadata)
    ZEND_PARSE_PARAMETERS_END();

    zval empty_initial;
    zval empty_trailing;
    if (initial_metadata == NULL) {
        array_init(&empty_initial);
        initial_metadata = &empty_initial;
    }
    if (trailing_metadata == NULL) {
        array_init(&empty_trailing);
        trailing_metadata = &empty_trailing;
    }

    zend_update_property_stringl(unary_result_ce, Z_OBJ_P(ZEND_THIS), "payload", sizeof("payload") - 1, ZSTR_VAL(payload), ZSTR_LEN(payload));
    zend_update_property(unary_result_ce, Z_OBJ_P(ZEND_THIS), "status", sizeof("status") - 1, status);
    zend_update_property(unary_result_ce, Z_OBJ_P(ZEND_THIS), "initialMetadata", sizeof("initialMetadata") - 1, initial_metadata);
    zend_update_property(unary_result_ce, Z_OBJ_P(ZEND_THIS), "trailingMetadata", sizeof("trailingMetadata") - 1, trailing_metadata);

    if (initial_metadata == &empty_initial) {
        zval_ptr_dtor(&empty_initial);
    }
    if (trailing_metadata == &empty_trailing) {
        zval_ptr_dtor(&empty_trailing);
    }
}

static const zend_function_entry channel_methods[] = {
    PHP_ME(FrankenGrpc_Channel, __construct, arginfo_channel_construct, ZEND_ACC_PUBLIC)
    PHP_ME(FrankenGrpc_Channel, close, arginfo_void, ZEND_ACC_PUBLIC)
    PHP_FE_END
};

static const zend_function_entry unary_call_methods[] = {
    PHP_ME(FrankenGrpc_UnaryCall, __construct, arginfo_unary_call_construct, ZEND_ACC_PUBLIC)
    PHP_ME(FrankenGrpc_UnaryCall, start, arginfo_unary_start, ZEND_ACC_PUBLIC)
    PHP_ME(FrankenGrpc_UnaryCall, cancel, arginfo_void, ZEND_ACC_PUBLIC)
    PHP_ME(FrankenGrpc_UnaryCall, getPeer, arginfo_get_peer, ZEND_ACC_PUBLIC)
    PHP_FE_END
};

static const zend_function_entry server_streaming_call_methods[] = {
    PHP_ME(FrankenGrpc_ServerStreamingCall, __construct, arginfo_stream_construct, ZEND_ACC_PUBLIC)
    PHP_ME(FrankenGrpc_ServerStreamingCall, start, arginfo_stream_start, ZEND_ACC_PUBLIC)
    PHP_ME(FrankenGrpc_ServerStreamingCall, read, arginfo_stream_read, ZEND_ACC_PUBLIC)
    PHP_ME(FrankenGrpc_ServerStreamingCall, getInitialMetadata, arginfo_metadata, ZEND_ACC_PUBLIC)
    PHP_ME(FrankenGrpc_ServerStreamingCall, getStatus, arginfo_status_return, ZEND_ACC_PUBLIC)
    PHP_ME(FrankenGrpc_ServerStreamingCall, getTrailingMetadata, arginfo_metadata, ZEND_ACC_PUBLIC)
    PHP_ME(FrankenGrpc_ServerStreamingCall, cancel, arginfo_void, ZEND_ACC_PUBLIC)
    PHP_ME(FrankenGrpc_ServerStreamingCall, getPeer, arginfo_get_peer, ZEND_ACC_PUBLIC)
    PHP_FE_END
};

static const zend_function_entry status_methods[] = {
    PHP_ME(FrankenGrpc_Status, __construct, arginfo_status_construct, ZEND_ACC_PUBLIC)
    PHP_FE_END
};

static const zend_function_entry unary_result_methods[] = {
    PHP_ME(FrankenGrpc_UnaryResult, __construct, arginfo_unary_result_construct, ZEND_ACC_PUBLIC)
    PHP_FE_END
};

PHP_MINIT_FUNCTION(frankengrpc)
{
    zend_class_entry ce;

    INIT_NS_CLASS_ENTRY(ce, "FrankenGrpc", "Channel", channel_methods);
    channel_ce = zend_register_internal_class(&ce);
    channel_ce->create_object = fg_channel_create;
    memcpy(&channel_handlers, zend_get_std_object_handlers(), sizeof(zend_object_handlers));
    channel_handlers.offset = XtOffsetOf(fg_handle_object, std);
    channel_handlers.free_obj = fg_channel_free;

    INIT_NS_CLASS_ENTRY(ce, "FrankenGrpc", "UnaryCall", unary_call_methods);
    unary_call_ce = zend_register_internal_class(&ce);
    unary_call_ce->create_object = fg_unary_call_create;
    memcpy(&unary_call_handlers, zend_get_std_object_handlers(), sizeof(zend_object_handlers));
    unary_call_handlers.offset = XtOffsetOf(fg_unary_call_object, std);
    unary_call_handlers.free_obj = fg_unary_call_free_obj;

    INIT_NS_CLASS_ENTRY(ce, "FrankenGrpc", "ServerStreamingCall", server_streaming_call_methods);
    server_streaming_call_ce = zend_register_internal_class(&ce);
    server_streaming_call_ce->create_object = fg_server_streaming_call_create;
    memcpy(&server_streaming_call_handlers, zend_get_std_object_handlers(), sizeof(zend_object_handlers));
    server_streaming_call_handlers.offset = XtOffsetOf(fg_handle_object, std);
    server_streaming_call_handlers.free_obj = fg_server_streaming_call_free_obj;

    INIT_NS_CLASS_ENTRY(ce, "FrankenGrpc", "Status", status_methods);
    status_ce = zend_register_internal_class(&ce);
    zend_declare_property_null(status_ce, "code", sizeof("code") - 1, ZEND_ACC_PUBLIC);
    zend_declare_property_null(status_ce, "details", sizeof("details") - 1, ZEND_ACC_PUBLIC);
    zend_declare_property_null(status_ce, "metadata", sizeof("metadata") - 1, ZEND_ACC_PUBLIC);

    INIT_NS_CLASS_ENTRY(ce, "FrankenGrpc", "UnaryResult", unary_result_methods);
    unary_result_ce = zend_register_internal_class(&ce);
    zend_declare_property_null(unary_result_ce, "payload", sizeof("payload") - 1, ZEND_ACC_PUBLIC);
    zend_declare_property_null(unary_result_ce, "status", sizeof("status") - 1, ZEND_ACC_PUBLIC);
    zend_declare_property_null(unary_result_ce, "initialMetadata", sizeof("initialMetadata") - 1, ZEND_ACC_PUBLIC);
    zend_declare_property_null(unary_result_ce, "trailingMetadata", sizeof("trailingMetadata") - 1, ZEND_ACC_PUBLIC);

    return SUCCESS;
}

zend_module_entry frankengrpc_module_entry = {
    STANDARD_MODULE_HEADER,
    "frankengrpc",
    NULL,
    PHP_MINIT(frankengrpc),
    NULL,
    NULL,
    NULL,
    NULL,
    "0.1.0",
    STANDARD_MODULE_PROPERTIES
};
