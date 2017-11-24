
#pragma once

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

extern int32_t sputnikvm_is_implemented(void);

enum {
    SPUTNIK_VM_CALL = 0,
    SPUTNIK_VM_CREATE = 1,

    SPUTNIK_VM_EXITED_OK = 0,
    SPUTNIK_VM_EXITED_ERR = 1,
    SPUTNIK_VM_RUNNING = 2,
    SPUTNIK_VM_UNSUPPORTED_ERR = 3,

    SPUTNIK_VM_REQUIRE_ACCOUNT = 2,
    SPUTNIK_VM_REQUIRE_CODE = 3,
    SPUTNIK_VM_REQUIRE_HASH = 4,
    SPUTNIK_VM_REQUIRE_VALUE = 5,
};

extern int32_t sputnikvm_fire(void *ctx);

extern void* sputnikvm_context(
    int32_t     callOrCreate,
    const char *gas,
    const char *price,
    const char *value,
    const char *caller,
    const char *target,
    const void *bytes,
    size_t      bytesLen,
    const char *gasLimit,
    const char *coinbase,
    int32_t     fork,
    const char *blocknum,
    uint64_t    time,
    const char *difficulty);

extern const char* sputnikvm_req_address(void *ctx);
extern const char* sputnikvm_req_hash(void *ctx);
extern uint64_t sputnikvm_req_blocknum(void *ctx);

extern void sputnikvm_commit_account(
    void *ctx,const char *address,uint64_t nonce,const char *balance,
    const void *bytes,size_t bytes_len);

extern void sputnikvm_commit_code(
    void *ctx,const char *address,const void *bytes,size_t bytes_len);

extern void sputnikvm_commit_value(
    void *ctx, const char *addressPtr, const char *keyPtr, const char *valuePtr);

extern void sputnikvm_commit_blockhash(void *ctx, uint64_t number, const char *hashPtr);

extern const void* sputnikvm_out(void *ctx);
extern size_t sputnikvm_out_len(void *ctx);
extern const char* sputnikvm_gas(void *ctx);
extern const char* sputnikvm_refund(void *ctx);
extern const char* sputnikvm_error(void *ctx);
extern int32_t sputnikvm_status(void *ctx);
extern void sputnikvm_terminate(void *ctx);
extern void* sputnikvm_first_account(void *ctx);
extern void* sputnikvm_next_account(void *ctx);
extern const char* sputnikvm_acc_address(void *ctx,void *acc);
extern const char* sputnikvm_acc_balance(void *ctx,void *acc);
extern int32_t sputnikvm_acc_change(void *acc);
extern uint64_t sputnikvm_acc_nonce(void *acc);
extern const void* sputnikvm_acc_code(void *ctx,void *acc);
extern size_t sputnikvm_acc_code_len(void *acc);
extern const char* sputnikvm_acc_first_key(void *ctx,void *acc);
extern const char* sputnikvm_acc_next_key(void *ctx,void *acc);
extern const char* sputnikvm_acc_value(void *ctx,void *acc,const char *key);
extern const char* sputnikvm_first_suicided(void *ctx);
extern const char* sputnikvm_next_suicided(void *ctx);
extern void* sputnikvm_first_log(void *ctx);
extern void* sputnikvm_next_log(void *ctx);
extern const char* sputnikvm_log_address(void *ctx,void *log);
extern const char* sputnikvm_log_topics(void *ctx,void *log);
extern const void* sputnikvm_log_data(void *ctx,void *log);
extern size_t sputnikvm_log_data_len(void *log);
