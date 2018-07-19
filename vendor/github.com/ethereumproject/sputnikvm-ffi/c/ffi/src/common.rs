use libc::{c_uchar};
use bigint::{U256, H256, Gas, Address};

// We use big-endian representation for c_u256 and c_gas. Note that
// however, in etcommon-bigint, it is little-endian representation.

#[repr(C)]
#[derive(Clone, Copy)]
pub struct c_address {
    pub data: [c_uchar; 20],
}

impl Into<Address> for c_address {
    fn into(self) -> Address {
        let mut a = [0u8; 20];
        for i in 0..20 {
            a[i] = self.data[i] as u8;
        }
        let b: &[u8] = &a;
        Address::from(b)
    }
}

impl Default for c_address {
    fn default() -> c_address {
        c_address {
            data: [0; 20]
        }
    }
}

impl From<Address> for c_address {
    fn from(val: Address) -> Self {
        let mut a = Self::default();
        for i in 0..20 {
            a.data[i] = val[i];
        }
        a
    }
}

#[repr(C)]
#[derive(Clone, Copy)]
pub struct c_gas {
    pub data: [c_uchar; 32],
}

impl Into<Gas> for c_gas {
    fn into(self) -> Gas {
        let u: c_u256 = c_u256 { data: self.data };
        let m: U256 = u.into();
        m.into()
    }
}

impl Default for c_gas {
    fn default() -> c_gas {
        c_gas {
            data: [0; 32]
        }
    }
}

impl From<Gas> for c_gas {
    fn from(val: Gas) -> Self {
        let m: U256 = val.into();
        let u: c_u256 = m.into();
        c_gas { data: u.data }
    }
}

#[repr(C)]
#[derive(Clone, Copy)]
pub struct c_u256 {
    pub data: [c_uchar; 32],
}

impl Into<U256> for c_u256 {
    fn into(self) -> U256 {
        let mut a = [0u8; 32];
        for i in 0..32 {
            a[i] = self.data[i] as u8;
        }
        let b: &[u8] = &a;
        U256::from(b)
    }
}

impl Default for c_u256 {
    fn default() -> c_u256 {
        c_u256 {
            data: [0; 32]
        }
    }
}

impl From<U256> for c_u256 {
    fn from(val: U256) -> Self {
        let mut a = Self::default();
        for i in 0..32 {
            a.data[i] = val.index(i) as c_uchar;
        }
        a
    }
}

#[repr(C)]
#[derive(Clone, Copy)]
pub struct c_h256 {
    pub data: [c_uchar; 32],
}

impl Into<H256> for c_h256 {
    fn into(self) -> H256 {
        let mut a = [0u8; 32];
        for i in 0..32 {
            a[i] = self.data[i] as u8;
        }
        let b: &[u8] = &a;
        H256::from(b)
    }
}

impl Default for c_h256 {
    fn default() -> c_h256 {
        c_h256 {
            data: [0; 32]
        }
    }
}

impl From<H256> for c_h256 {
    fn from(val: H256) -> Self {
        let mut a = Self::default();
        for i in 0..32 {
            a.data[i] = val[i] as c_uchar;
        }
        a
    }
}

