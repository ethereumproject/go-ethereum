extern crate libc;
extern crate bigint;
extern crate sputnikvm;

mod common;

pub use common::{c_address, c_gas, c_u256, c_h256};

use std::slice;
use std::ptr;
use std::rc::Rc;
use std::ops::DerefMut;
use std::collections::HashMap;
use libc::{c_uchar, c_uint, c_longlong};
use bigint::{U256, M256};
use sputnikvm::{TransactionAction, ValidTransaction, HeaderParams, SeqTransactionVM, Patch,
                MainnetFrontierPatch, MainnetHomesteadPatch, MainnetEIP150Patch, MainnetEIP160Patch,
                VM, VMStatus, RequireError, AccountCommitment, AccountChange,
                FrontierPatch, HomesteadPatch, EIP150Patch, EIP160Patch, AccountPatch};

type c_action = c_uchar;
#[no_mangle]
pub static CALL_ACTION: c_action = 0;
#[no_mangle]
pub static CREATE_ACTION: c_action = 1;

pub struct MordenAccountPatch;
impl AccountPatch for MordenAccountPatch {
    fn initial_nonce() -> U256 { U256::from(1048576) }
}

pub type MordenFrontierPatch = FrontierPatch<MordenAccountPatch>;
pub type MordenHomesteadPatch = HomesteadPatch<MordenAccountPatch>;
pub type MordenEIP150Patch = EIP150Patch<MordenAccountPatch>;
pub type MordenEIP160Patch = EIP160Patch<MordenAccountPatch>;

static mut CUSTOM_INITIAL_NONCE: Option<U256> = None;

pub struct CustomAccountPatch;
impl AccountPatch for CustomAccountPatch {
    fn initial_nonce() -> U256 { U256::from(unsafe { CUSTOM_INITIAL_NONCE.unwrap() }) }
}

pub type CustomFrontierPatch = FrontierPatch<CustomAccountPatch>;
pub type CustomHomesteadPatch = HomesteadPatch<CustomAccountPatch>;
pub type CustomEIP150Patch = EIP150Patch<CustomAccountPatch>;
pub type CustomEIP160Patch = EIP160Patch<CustomAccountPatch>;

#[repr(C)]
pub struct c_transaction {
    pub caller: c_address,
    pub gas_price: c_gas,
    pub gas_limit: c_gas,
    pub action: c_action,
    pub action_address: c_address,
    pub value: c_u256,
    pub input: *const c_uchar,
    pub input_len: c_uint,
    pub nonce: c_u256,
}

#[repr(C)]
pub struct c_header_params {
    pub beneficiary: c_address,
    pub timestamp: c_longlong,
    pub number: c_u256,
    pub difficulty: c_u256,
    pub gas_limit: c_gas,
}

#[repr(C)]
pub struct c_require {
    pub typ: c_require_type,
    pub value: c_require_value,
}

#[repr(C)]
pub enum c_require_type {
    none,
    account,
    account_code,
    account_storage,
    blockhash
}

#[repr(C)]
pub union c_require_value {
    pub account: c_address,
    pub account_storage: c_require_value_account_storage,
    pub blockhash: c_u256,
}

#[repr(C)]
#[derive(Clone, Copy)]
pub struct c_require_value_account_storage {
    pub address: c_address,
    pub key: c_u256,
}

#[repr(C)]
pub struct c_log {
    pub address: c_address,
    pub topic_len: c_uint,
    pub data_len: c_uint,
}

#[repr(C)]
pub struct c_account_change {
    pub typ: c_account_change_type,
    pub value: c_account_change_value,
}

#[repr(C)]
pub enum c_account_change_type {
    increase_balance,
    decrease_balance,
    full,
    create,
    removed,
}

#[repr(C)]
pub union c_account_change_value {
    pub balance: c_account_change_value_balance,
    pub all: c_account_change_value_all,
    pub removed: c_address,
}

#[repr(C)]
#[derive(Clone, Copy)]
pub struct c_account_change_value_balance {
    pub address: c_address,
    pub amount: c_u256,
}

#[repr(C)]
#[derive(Clone, Copy)]
pub struct c_account_change_value_all {
    pub address: c_address,
    pub nonce: c_u256,
    pub balance: c_u256,
    pub storage_len: c_uint,
    pub code_len: c_uint,
}

#[repr(C)]
pub struct c_account_change_storage {
    pub key: c_u256,
    pub value: c_u256,
}

#[no_mangle]
pub extern "C" fn print_u256(v: c_u256) {
    let v: U256 = v.into();
    println!("{}", v);
}

#[no_mangle]
pub unsafe extern "C" fn sputnikvm_set_custom_initial_nonce(v: c_u256) {
    let v: U256 = v.into();
    unsafe {
        CUSTOM_INITIAL_NONCE = Some(v)
    }
}

fn sputnikvm_new<P: Patch + 'static>(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    let transaction = ValidTransaction {
        caller: Some(transaction.caller.into()),
        gas_price: transaction.gas_price.into(),
        gas_limit: transaction.gas_limit.into(),
        action: if transaction.action == CALL_ACTION {
            TransactionAction::Call(transaction.action_address.into())
        } else if transaction.action == CREATE_ACTION {
            TransactionAction::Create
        } else {
            panic!()
        },
        value: transaction.value.into(),
        input: {
            if transaction.input.is_null() {
                Rc::new(Vec::new())
            } else {
                let s = unsafe {
                    slice::from_raw_parts(transaction.input, transaction.input_len as usize)
                };
                let mut r = Vec::new();
                for v in s {
                    r.push(*v);
                }
                Rc::new(r)
            }
        },
        nonce: transaction.nonce.into(),
    };

    let header = HeaderParams {
        beneficiary: header.beneficiary.into(),
        timestamp: header.timestamp as u64,
        number: header.number.into(),
        difficulty: header.difficulty.into(),
        gas_limit: header.gas_limit.into(),
    };

    let vm = SeqTransactionVM::<P>::new(transaction, header);
    Box::into_raw(Box::new(Box::new(vm)))
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_frontier(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<MainnetFrontierPatch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_homestead(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<MainnetHomesteadPatch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_eip150(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<MainnetEIP150Patch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_eip160(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<MainnetEIP160Patch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_morden_frontier(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<MordenFrontierPatch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_morden_homestead(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<MordenHomesteadPatch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_morden_eip150(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<MordenEIP150Patch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_morden_eip160(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<MordenEIP160Patch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_custom_frontier(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<CustomFrontierPatch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_custom_homestead(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<CustomHomesteadPatch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_custom_eip150(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<CustomEIP150Patch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_new_custom_eip160(
    transaction: c_transaction, header: c_header_params
) -> *mut Box<VM> {
    sputnikvm_new::<CustomEIP160Patch>(transaction, header)
}

#[no_mangle]
pub extern "C" fn sputnikvm_free(
    vm: *mut Box<VM>
) {
    if vm.is_null() { return; }
    unsafe { Box::from_raw(vm); }
}

#[no_mangle]
pub extern "C" fn sputnikvm_fire(
    vm: *mut Box<VM>
) -> c_require {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let ret;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        match vm.fire() {
            Ok(()) => {
                ret = c_require {
                    typ: c_require_type::none,
                    value: c_require_value {
                        account: c_address::default(),
                    }
                };
            },
            Err(RequireError::Account(address)) => {
                ret = c_require {
                    typ: c_require_type::account,
                    value: c_require_value {
                        account: address.into(),
                    }
                };
            },
            Err(RequireError::AccountCode(address)) => {
                ret = c_require {
                    typ: c_require_type::account_code,
                    value: c_require_value {
                        account: address.into(),
                    }
                };
            },
            Err(RequireError::AccountStorage(address, key)) => {
                ret = c_require {
                    typ: c_require_type::account_storage,
                    value: c_require_value {
                        account_storage: c_require_value_account_storage {
                            address: address.into(),
                            key: key.into(),
                        },
                    }
                };
            },
            Err(RequireError::Blockhash(number)) => {
                ret = c_require {
                    typ: c_require_type::blockhash,
                    value: c_require_value {
                        blockhash: number.into(),
                    },
                };
            },
        }
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_commit_account(
    vm: *mut Box<VM>, address: c_address, nonce: c_u256, balance: c_u256,
    code: *mut c_uchar, code_len: c_uint
) -> bool {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let ret;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        let commitment = AccountCommitment::Full {
            nonce: nonce.into(),
            address: address.into(),
            balance: balance.into(),
            code: {
                let code = unsafe { slice::from_raw_parts(code, code_len as usize) };
                Rc::new(code.into())
            },
        };
        match vm.commit_account(commitment) {
            Ok(()) => { ret = true; }
            Err(_) => { ret = false; }
        }
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_commit_account_code(
    vm: *mut Box<VM>, address: c_address, code: *mut c_uchar, code_len: c_uint
) -> bool {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let ret;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        let commitment = AccountCommitment::Code {
            address: address.into(),
            code: {
                let code = unsafe { slice::from_raw_parts(code, code_len as usize) };
                Rc::new(code.into())
            },
        };
        match vm.commit_account(commitment) {
            Ok(()) => { ret = true; }
            Err(_) => { ret = false; }
        }
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_commit_account_storage(
    vm: *mut Box<VM>, address: c_address, index: c_u256, value: c_u256
) -> bool {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let ret;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        let commitment = AccountCommitment::Storage {
            address: address.into(),
            index: index.into(),
            value: {
                let value: U256 = value.into();
                value.into()
            },
        };
        match vm.commit_account(commitment) {
            Ok(()) => { ret = true; }
            Err(_) => { ret = false; }
        }
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_commit_nonexist(
    vm: *mut Box<VM>, address: c_address
) -> bool {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let ret;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        let commitment = AccountCommitment::Nonexist(address.into());
        match vm.commit_account(commitment) {
            Ok(()) => { ret = true; }
            Err(_) => { ret = false; }
        }
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_commit_blockhash(
    vm: *mut Box<VM>, number: c_u256, hash: c_h256
) -> bool {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let ret;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        match vm.commit_blockhash(number.into(), hash.into()) {
            Ok(()) => { ret = true; }
            Err(_) => { ret = false; }
        }
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_logs_len(
    vm: *mut Box<VM>
) -> c_uint {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let ret;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        ret = vm.logs().len() as c_uint;
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_logs_copy_info(
    vm: *mut Box<VM>, log: *mut c_log, log_len: c_uint
) {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        let logs = vm.logs();
        let mut logs_write = unsafe { slice::from_raw_parts_mut(log, log_len as usize) };
        for i in 0..logs_write.len() {
            if i < logs.len() {
                logs_write[i] = c_log {
                    address: logs[i].address.into(),
                    topic_len: logs[i].topics.len() as c_uint,
                    data_len: logs[i].data.len() as c_uint,
                };
            }
        }
    }
    Box::into_raw(vm_box);
}

#[no_mangle]
pub extern "C" fn sputnikvm_logs_topic(
    vm: *mut Box<VM>, log_index: c_uint, topic_index: c_uint
) -> c_h256 {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let ret;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        ret = vm.logs()[log_index as usize].topics[topic_index as usize].into();
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_logs_copy_data(
    vm: *mut Box<VM>, log_index: c_uint, data_w: *mut u8, data_w_len: c_uint
) {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        let logs = vm.logs();
        let mut data_w = unsafe { slice::from_raw_parts_mut(data_w, data_w_len as usize) };
        for i in 0..data_w.len() {
            if i < logs[log_index as usize].data.len() {
                data_w[i] = logs[log_index as usize].data[i];
            }
        }
    }
    Box::into_raw(vm_box);
}

#[no_mangle]
pub extern "C" fn sputnikvm_account_changes_len(
    vm: *mut Box<VM>
) -> c_uint {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let ret;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        ret = vm.accounts().len() as c_uint;
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_account_changes_copy_info(
    vm: *mut Box<VM>, w: *mut c_account_change, wl: c_uint
) {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        let accounts = vm.accounts();
        let mut w = unsafe { slice::from_raw_parts_mut(w, wl as usize) };
        for (i, account) in accounts.enumerate() {
            if i < w.len() {
                w[i] = match account {
                    &AccountChange::Full { nonce, address, balance, ref changing_storage, ref code } => {
                        c_account_change {
                            typ: c_account_change_type::full,
                            value: c_account_change_value {
                                all: c_account_change_value_all {
                                    address: address.into(),
                                    nonce: nonce.into(),
                                    balance: balance.into(),
                                    storage_len: changing_storage.len() as c_uint,
                                    code_len: code.len() as c_uint,
                                },
                            },
                        }
                    },
                    &AccountChange::Create { nonce, address, balance, ref storage, ref code } => {
                        c_account_change {
                            typ: c_account_change_type::create,
                            value: c_account_change_value {
                                all: c_account_change_value_all {
                                    address: address.into(),
                                    nonce: nonce.into(),
                                    balance: balance.into(),
                                    storage_len: storage.len() as c_uint,
                                    code_len: code.len() as c_uint,
                                },
                            },
                        }
                    },
                    &AccountChange::Nonexist(address) => {
                        c_account_change {
                            typ: c_account_change_type::removed,
                            value: c_account_change_value {
                                removed: address.into(),
                            },
                        }
                    },
                    &AccountChange::IncreaseBalance(address, amount) => {
                        c_account_change {
                            typ: c_account_change_type::increase_balance,
                            value: c_account_change_value {
                                balance: c_account_change_value_balance {
                                    address: address.into(),
                                    amount: amount.into(),
                                },
                            }
                        }
                    },
                    &AccountChange::DecreaseBalance(address, amount) => {
                        c_account_change {
                            typ: c_account_change_type::decrease_balance,
                            value: c_account_change_value {
                                balance: c_account_change_value_balance {
                                    address: address.into(),
                                    amount: amount.into(),
                                },
                            }
                        }
                    },
                }
            }
        }
    }
    Box::into_raw(vm_box);
}

#[no_mangle]
pub extern "C" fn sputnikvm_account_changes_copy_storage(
    vm: *mut Box<VM>, address: c_address, w: *mut c_account_change_storage, wl: c_uint
) -> bool {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let mut ret = false;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        let accounts = vm.accounts();
        let mut w = unsafe { slice::from_raw_parts_mut(w, wl as usize) };
        let target_address = address.into();
        for account in accounts {
            match account {
                &AccountChange::Full { address, ref changing_storage, .. } => {
                    if address == target_address {
                        let storage: HashMap<U256, M256> = changing_storage.clone().into();
                        for (i, (key, value)) in storage.iter().enumerate() {
                            if i < w.len() {
                                w[i] = c_account_change_storage {
                                    key: (*key).into(),
                                    value: {
                                        let u: U256 = (*value).into();
                                        u.into()
                                    }
                                }
                            }
                        }
                        ret = true;
                        break;
                    }
                },
                &AccountChange::Create { address, ref storage, .. } => {
                    if address == target_address {
                        let storage: HashMap<U256, M256> = storage.clone().into();
                        for (i, (key, value)) in storage.iter().enumerate() {
                            if i < w.len() {
                                w[i] = c_account_change_storage {
                                    key: (*key).into(),
                                    value: {
                                        let u: U256 = (*value).into();
                                        u.into()
                                    }
                                }
                            }
                        }
                        ret = true;
                        break;
                    }
                },
                _ => {},
            }
        }
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_account_changes_copy_code(
    vm: *mut Box<VM>, address: c_address, w: *mut u8, wl: c_uint
) -> bool {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let mut ret = false;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        let accounts = vm.accounts();
        let mut w = unsafe { slice::from_raw_parts_mut(w, wl as usize) };
        let target_address = address.into();
        for account in accounts {
            match account {
                &AccountChange::Full { address, ref code, .. } => {
                    if address == target_address {
                        for i in 0..w.len() {
                            if i < code.len() {
                                w[i] = code[i];
                            }
                        }
                        ret = true;
                        break;
                    }
                },
                &AccountChange::Create { address, ref code, .. } => {
                    if address == target_address {
                        for i in 0..w.len() {
                            if i < code.len() {
                                w[i] = code[i];
                            }
                        }
                        ret = true;
                        break;
                    }
                },
                _ => {},
            }
        }
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_used_gas(vm: *mut Box<VM>) -> c_gas {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let ret;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        ret = vm.used_gas().into();
    }
    Box::into_raw(vm_box);
    ret
}

#[no_mangle]
pub extern "C" fn sputnikvm_default_transaction() -> c_transaction {
    c_transaction {
        caller: c_address::default(),
        gas_price: c_gas::default(),
        gas_limit: c_gas::default(),
        action: CALL_ACTION,
        action_address: c_address::default(),
        value: c_u256::default(),
        input: ptr::null(),
        input_len: 0,
        nonce: c_u256::default(),
    }
}

#[no_mangle]
pub extern "C" fn sputnikvm_default_header_params() -> c_header_params {
    c_header_params {
        beneficiary: c_address::default(),
        timestamp: 0,
        number: c_u256::default(),
        difficulty: c_u256::default(),
        gas_limit: c_gas::default(),
    }
}

#[no_mangle]
pub extern "C" fn sputnikvm_status_failed(vm: *mut Box<VM>) -> c_uchar {
    let mut vm_box = unsafe { Box::from_raw(vm) };
    let ret;
    {
        let vm: &mut VM = vm_box.deref_mut().deref_mut();
        match vm.status() {
            VMStatus::ExitedErr(_) => ret = 1,
            default => ret = 0,
        }
    }
    Box::into_raw(vm_box);
    ret
}
