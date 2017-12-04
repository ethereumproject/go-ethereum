// Copyright 2017 (c) ETCDEV Team
//
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

#pragma once

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

extern int32_t sputnikvm_is_implemented(void);

enum {
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
    const void *gas,
    const void *price,
    const void *value,
    const void *caller,
    const void *target,
    const void *bytes,
    size_t      bytesLen,
    const void *gasLimit,
    const void *coinbase,
    int32_t     fork,
    uint64_t    blocknum,
    uint64_t    time,
    const void *difficulty);

extern const char* sputnikvm_req_address(void *ctx);
extern const char* sputnikvm_req_hash(void *ctx);
extern uint64_t sputnikvm_req_blocknum(void *ctx);

extern void sputnikvm_commit_value(
    void *ctx,
    const void *address,
    const void *key,
    const void *value);

extern void sputnikvm_commit_account(
    void *ctx,
    const void *address,
    uint64_t nonce,
    const void *balance,
    const void *code,size_t code_len);

extern void sputnikvm_commit_code(
    void *ctx,
    const void *address,
    const void *code,size_t code_len);

extern void sputnikvm_commit_blockhash(
    void *ctx,
    uint64_t number,
    const void *hash);

extern const char* sputnikvm_error(void *ctx);
extern int32_t sputnikvm_status(void *ctx);
extern void sputnikvm_terminate(void *ctx);

extern size_t sputnikvm_gas_copy(void *ctx, void *bits);
extern size_t sputnikvm_refund_copy(void *ctx, void *bits);
extern size_t sputnikvm_out_len(void *ctx);
extern size_t sputnikvm_out_copy(void *ctx, void *out);
extern size_t sputnikvm_req_address_copy(void *ctx, void *address);
extern size_t sputnikvm_req_hash_copy(void *ctx, void *hash);

extern void* sputnikvm_first_account(void *ctx);
extern void* sputnikvm_next_account(void *ctx);
extern int32_t sputnikvm_acc_change(void *acc);
extern uint64_t sputnikvm_acc_nonce(void *acc);
extern size_t sputnikvm_acc_balance_copy(void *acc, void *bits);
extern size_t sputnikvm_acc_address_copy(void *acc, void *address);
extern size_t sputnikvm_acc_code_len(void *acc);
extern size_t sputnikvm_acc_code_copy(void *acc, void *code);
extern size_t sputnikvm_acc_first_kv_copy(void *ctx, void *acc, void *key, void *val);
extern size_t sputnikvm_acc_next_kv_copy(void *ctx, void *key, void *val);

extern void* sputnikvm_log(void *ctx, size_t index);
extern size_t sputnikvm_logs_count(void *ctx);
extern size_t sputnikvm_log_address_copy(void *log, void *address);
extern size_t sputnikvm_log_data_len(void *log);
extern size_t sputnikvm_log_data_copy(void *log, void *data);
extern size_t sputnikvm_log_topics_count(void *log);
extern size_t sputnikvm_log_topic_copy(void *log, size_t index, void *topic);

extern size_t sputnikvm_suicides_count(void *ctx);
extern size_t sputnikvm_suicide_copy(void *ctx, size_t index, void *address);
