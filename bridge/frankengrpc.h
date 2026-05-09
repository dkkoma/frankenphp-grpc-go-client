#ifndef FRANKENGRPC_H
#define FRANKENGRPC_H

#include <php.h>
#include <stdint.h>

typedef struct {
    char *data;
    size_t len;
} fg_string;

typedef struct {
    fg_string *values;
    size_t values_len;
    char *key;
    size_t key_len;
} fg_metadata_entry;

typedef struct {
    fg_metadata_entry *entries;
    size_t entries_len;
} fg_metadata;

typedef struct {
    int code;
    char *details;
    size_t details_len;
    fg_metadata metadata;
} fg_status;

typedef struct {
    int has_error;
    char *error;
    size_t error_len;
} fg_error;

typedef struct {
    fg_error error;
    fg_string payload;
    fg_status status;
    fg_metadata initial_metadata;
    fg_metadata trailing_metadata;
    char *peer;
    size_t peer_len;
} fg_unary_result;

typedef struct {
    fg_error error;
    int has_message;
    fg_string payload;
} fg_stream_read_result;

extern zend_module_entry frankengrpc_module_entry;

/* C includes _cgo_export.h for exported Go symbols. */

#endif
