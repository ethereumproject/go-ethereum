/** @file ext.c
 * Ethereum extensions to libecp256k1
 * @authors:
 *   Arkadiy Paronyan <arkady@ethdev.com>
 * @date 2015
 */

#include "src/secp256k1.c"

int secp256k1_ecdh_raw(const secp256k1_context* ctx, unsigned char *result, const secp256k1_pubkey *point, const unsigned char *scalar)
{
	int ret = 0;
	int overflow = 0;
	secp256k1_gej res;
	secp256k1_ge pt;
	secp256k1_scalar s;
	ARG_CHECK(result != NULL);
	ARG_CHECK(point != NULL);
	ARG_CHECK(scalar != NULL);

	secp256k1_pubkey_load(ctx, &pt, point);
	secp256k1_scalar_set_b32(&s, scalar, &overflow);
	if (overflow || secp256k1_scalar_is_zero(&s))
		ret = 0;
	else
	{
		secp256k1_ecmult_const(&res, &pt, &s);
		secp256k1_ge_set_gej(&pt, &res);
		secp256k1_fe_normalize(&pt.x);
		secp256k1_fe_normalize(&pt.y);
		secp256k1_fe_get_b32(result, &pt.x);
		ret = 1;
	}

	secp256k1_scalar_clear(&s);
	return ret;
}

/// Returns inverse (1 / n) of secret key `seckey`
int secp256k1_ec_privkey_inverse(const secp256k1_context* ctx, unsigned char *inversed, const unsigned char* seckey) {
	secp256k1_scalar inv;
	secp256k1_scalar sec;
	int ret = 0;
	int overflow = 0;
	VERIFY_CHECK(ctx != NULL);
	ARG_CHECK(inversed != NULL);
	ARG_CHECK(seckey != NULL);

	secp256k1_scalar_set_b32(&sec, seckey, NULL);
	ret = !overflow;
	if (ret) {
		memset(inversed, 0, 32);
		secp256k1_scalar_inverse(&inv, &sec);
		secp256k1_scalar_get_b32(inversed, &inv);
	}

	secp256k1_scalar_clear(&inv);
	secp256k1_scalar_clear(&sec);
	return ret;
}
