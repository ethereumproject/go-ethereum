
#pragma once

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

extern int32_t sputnikvm_is_implemented(void);

enum {
   SPUTNIK_VM_CALL = 0,
   SPUTNIK_VM_CREATE = 1,

   SPUTNIK_VM_EXITED = 0,
   SPUTNIK_VM_REQUIRE_ACCOUNT = 2,
   SPUTNIK_VM_REQUIRE_CODE = 3,
   SPUTNIK_VM_REQUIRE_HASH = 4,
   SPUTNIK_VM_REQUIRE_VALUE = 5,
};

extern int32_t sputnikvm_fire(void *ctx);

extern const void* sputnikvm_context(
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
