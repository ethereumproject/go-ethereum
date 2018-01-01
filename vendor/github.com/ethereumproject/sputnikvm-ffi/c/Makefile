build:
	cd ffi && cargo build --release
	cp ffi/target/release/libsputnikvm_ffi.a libsputnikvm.a
	cp ffi/target/release/libsputnikvm_ffi.so libsputnikvm.so

debug:
	cd ffi && cargo build
	cp ffi/target/debug/libsputnikvm_ffi.a libsputnikvm.a
	cp ffi/target/debug/libsputnikvm_ffi.so libsputnikvm.so
