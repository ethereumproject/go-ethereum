/**
 * @file sputnikvm.h
 * @author Wei Tang
 * @brief SputnikVM FFI Bindings
 */

/**
 * 160-bit address.
 */
typedef struct {
  unsigned char data[20];
} sputnikvm_address;

/**
 * 256-bit integer for tracking gas usage.
 */
typedef struct {
  unsigned char data[32]; /** Big-endian aligned raw integer value. */
} sputnikvm_gas;

/**
 * Unsigned 256-bit integer.
 */
typedef struct {
  unsigned char data[32]; /** Big-endian aligned raw integer value. */
} sputnikvm_u256;

/**
 * 256-bit hash.
 */
typedef struct {
  unsigned char data[32];
} sputnikvm_h256;

extern void
print_u256(sputnikvm_u256 v);

/**
 * Action item used in a transaction, can be either CALL_ACTION or
 * CREATE_ACTION.
 */
typedef unsigned char sputnikvm_action;
extern const unsigned char CALL_ACTION;
extern const unsigned char CREATE_ACTION;

/**
 * Represents a valid EVM transaction. Used when creating a new VM
 * instance.
 */
typedef struct {
  sputnikvm_address caller;
  sputnikvm_gas gas_price;
  sputnikvm_gas gas_limit;
  sputnikvm_action action;
  sputnikvm_address action_address;
  sputnikvm_u256 value;
  unsigned char *input;
  unsigned int input_len;
  sputnikvm_u256 nonce;
} sputnikvm_transaction;

/**
 * Header parameters used when creating a new VM instance.
 */
typedef struct {
  sputnikvm_address beneficiary;
  unsigned long long int timestamp;
  sputnikvm_u256 number;
  sputnikvm_u256 difficulty;
  sputnikvm_gas gas_limit;
} sputnikvm_header_params;

typedef enum {
  require_none, require_account, require_account_code, require_account_storage, require_blockhash
} sputnikvm_require_type;

typedef struct {
  sputnikvm_address address;
  sputnikvm_u256 key;
} sputnikvm_require_value_account_storage;

typedef union {
  sputnikvm_address account;
  sputnikvm_require_value_account_storage account_storage;
  sputnikvm_u256 blockhash;
} sputnikvm_require_value;

typedef struct {
  sputnikvm_require_type typ;
  sputnikvm_require_value value;
} sputnikvm_require;

typedef struct {
  sputnikvm_address address;
  unsigned int topic_len;
  unsigned int data_len;
} sputnikvm_log;

typedef enum {
  account_change_increase_balance, account_change_decrease_balance, account_change_full, account_change_create, account_change_removed
} sputnikvm_account_change_type;

typedef struct {
  sputnikvm_address address;
  sputnikvm_u256 amount;
} sputnikvm_account_change_value_balance;

typedef struct {
  sputnikvm_address address;
  sputnikvm_u256 nonce;
  sputnikvm_u256 balance;
  unsigned int storage_len;
  unsigned int code_len;
} sputnikvm_account_change_value_all;

typedef union {
  sputnikvm_account_change_value_balance balance;
  sputnikvm_account_change_value_all all;
  sputnikvm_address removed;
} sputnikvm_account_change_value;

typedef struct {
  sputnikvm_account_change_type typ;
  sputnikvm_account_change_value value;
} sputnikvm_account_change;

typedef struct {
  sputnikvm_u256 key;
  sputnikvm_u256 value;
} sputnikvm_account_change_storage;

typedef struct sputnikvm_vm_S sputnikvm_vm_t;

/**
 * Create a new frontier patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_frontier(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Create a new homestead patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_homestead(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Create a new EIP150 patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_eip150(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Create a new EIP160 patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_eip160(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Create a new frontier morden patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_morden_frontier(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Create a new homestead morden patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_morden_homestead(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Create a new EIP150 morden patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_morden_eip150(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Create a new EIP160 morden patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_morden_eip160(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Create a new frontier custom patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_custom_frontier(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Create a new homestead custom patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_custom_homestead(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Create a new EIP150 custom patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_custom_eip150(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Create a new EIP160 custom patch EVM instance using the given
 * transaction and header parameters.
 */
extern sputnikvm_vm_t *
sputnikvm_new_custom_eip160(sputnikvm_transaction transaction, sputnikvm_header_params header);

/**
 * Set the initial nonce value for custom patch.
 */
extern void
sputnikvm_set_custom_initial_nonce(sputnikvm_u256 nonce);

/**
 * Execute the VM until it reaches a require error.
 */
extern sputnikvm_require
sputnikvm_fire(sputnikvm_vm_t *vm);

/**
 * Free a VM instance.
 */
extern void
sputnikvm_free(sputnikvm_vm_t *vm);

/**
 * Commit a full account value into the VM. Should be used together
 * with RequireError.
 */
extern int
sputnikvm_commit_account(sputnikvm_vm_t *vm, sputnikvm_address address, sputnikvm_u256 nonce, sputnikvm_u256 balance, unsigned char *code, unsigned int code_len);

/**
 * Commit a partial account code value into the VM. Should be used
 * together with RequireError.
 */
extern int
sputnikvm_commit_account_code(sputnikvm_vm_t *vm, sputnikvm_address address, unsigned char *code, unsigned int code_len);

/**
 * Commit a single account storage key-value pair into the VM. Should
 * be used together with RequireError.
 */
extern int
sputnikvm_commit_account_storage(sputnikvm_vm_t *vm, sputnikvm_address address, sputnikvm_u256 key, sputnikvm_u256 value);

/**
 * Mark a given required account as not-existing. Should be used
 * together with RequireError.
 */
extern int
sputnikvm_commit_nonexist(sputnikvm_vm_t *vm, sputnikvm_address address);

/**
 * Commit a block hash value with the specified block number. Should
 * be used together with RequireError.
 */
extern int
sputnikvm_commit_blockhash(sputnikvm_vm_t *vm, sputnikvm_u256 number, sputnikvm_h256 hash);

/**
 * Return the length of the logs after the VM has exited.
 */
extern unsigned int
sputnikvm_logs_len(sputnikvm_vm_t *vm);

/**
 * Copy the appended VM logs information after the VM has exited.
 */
extern void
sputnikvm_logs_copy_info(sputnikvm_vm_t *vm, sputnikvm_log *log, unsigned int log_len);

/**
 * Get the given VM logs topic. The log_index and topic_index must be
 * within the limit fetched from logs_len and logs_info.
 */
extern sputnikvm_h256
sputnikvm_logs_topic(sputnikvm_vm_t *vm, unsigned int log_index, unsigned int topic_index);

/**
 * Copy the data field of the given log.
 */
extern void
sputnikvm_logs_copy_data(sputnikvm_vm_t *vm, unsigned int log_index, unsigned char *data, unsigned int data_len);

/**
 * Get the account change length after the VM has exited.
 */
extern unsigned int
sputnikvm_account_changes_len(sputnikvm_vm_t *vm);

/**
 * Copy account change information.
 */
extern void
sputnikvm_account_changes_copy_info(sputnikvm_vm_t *vm, sputnikvm_account_change *w, unsigned int len);

/**
 * Copy storage value for a single account change entry. Note that
 * storage values are unordered.
 */
extern int
sputnikvm_account_changes_copy_storage(sputnikvm_vm_t *vm, sputnikvm_address address, sputnikvm_account_change_storage *w, unsigned int len);

/**
 * Copy code for a single account change entry.
 */
extern int
sputnikvm_account_changes_copy_code(sputnikvm_vm_t *vm, sputnikvm_address address, unsigned char *w, unsigned int len);

/**
 * Return the used gas after the VM has exited.
 */
extern sputnikvm_gas
sputnikvm_used_gas(sputnikvm_vm_t *vm);

/**
 * Default all-zero transaction value.
 */
extern sputnikvm_transaction
sputnikvm_default_transaction(void);

/**
 * Default all-zero header parameter value.
 */
extern sputnikvm_header_params
sputnikvm_default_header_params(void);

/**
 * Returns 1 if VM failed (VMStatus::ExitedErr), 0 otherwise (including VM is still running).
 */
extern char
sputnikvm_status_failed(sputnikvm_vm_t *vm);
