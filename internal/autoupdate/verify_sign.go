// Package autoupdate provides signature verification for the JFrog credential provider auto-update process.
// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package autoupdate

import (
	"bytes"
	"jfrog-credential-provider/internal/logger"
	"os"
	"strings"

	"golang.org/x/crypto/openpgp"
)

const publicKey = `
-----BEGIN PGP PUBLIC KEY BLOCK-----

mQINBGg+x1sBEADJxcIFZYF0DsgFaK2FXqmYJbTwkGuG59eXMfQnASrCX8GoF6sf
h4sgpLBEvwHDE7WdL5gX/kKiQcp8E4GPt4k7Huq1odWj/gd/b+KGFRxNlp+Gp03E
rxBf4ZYZ5MGIN1eMeG5fEqNFvcuDjROq8kmBTXVMxoUME622Ka4TtX47Mo4roxHe
m6kgOHBfHNIEGLAmjsg8BXtacnFvB05qv881m3kz6zxS6l4LaBbeLRo2niu/kAf1
88Mmu0WJuoRDu8nNND4dnvJOKm7boi/0kqXZx3Uh9ypFvjQqF91UcQter7jei8Je
lyyvhHG1nPO32Y0gTHH3dqplh34dDrBaNAsRcon1vWtMFboAtvohkLnymvjKL3EE
/39kwULZkklWeIRd12xTomK64pPdjWBwaadK3en6MjP3fVlKSN8Cu9yF4gN8N1ky
+2Hx2+GMUrc5EnTdrmHfTkDsXbLezwmXwvycUu44GecDglYcdiFUsmZsK2qv2XvL
Whjsn2Yoom74HKob6aV6ZaQNzBW/vs1yRCQrfqFgyHHKibbL21zMLYbd2xY1jSZM
oJUMKYclsMI7aXhg6+qN9G5CVPmQ4N3L0GwuXYuOabwhuqzOLo6jolHvPxseAKTP
XDCj1noEkXIaM7pbhG94lDxqbVETmMaDRenqpmAGZhjYqpgZaXghyUQomQARAQAB
tEhKZnJvZyAoSmZyb2cgS3ViZWxldCBDcmVkZW50aWFsIFByb3ZpZGVyIFBsdWdp
biBHUEcpIDxzdXBwb3J0QGpmcm9nLmNvbT6JAlcEEwEIAEEWIQTe3l0eHi28VZY9
jax1OiSNfuvq7gUCaD7HWwIbAwUJA8JnAAULCQgHAgIiAgYVCgkICwIEFgIDAQIe
BwIXgAAKCRB1OiSNfuvq7nBlD/4uyhRMuLcQbesicOdgp9tNn+uLWCZ3QJQR0/ck
TJQ57VTkif4IJVSd6llirKirnh1wvD8WllLeJVkR68kq6Mfd0jt2ArJoTH37ADS7
3dFRCM8pAwv23TfUM+FcwL3xKqbWS2vWaRA5NsR4ScbL9lBeQcJRshnxFtIPt7J9
mKsuYSsQqfSDsx+Kjphq1Xe/1YtIiKAuDiUcyP3tX0U7tjg7UjW+MkODo3c7ClI+
+4aurXdOMNZViCnFV4Lkpu1kQQMQD/6PdB29aKC5UOsZfGM0qOyOE4MzeANL/ALg
S666dj5+dzE8vcERR6589ylTY3/m8rS0aan84IWKXqagXEdSQq4jve7+TCAHFg+S
3Jjvgp4RryUvo31sy6ct4wGKWlQ06cVHDlRhnrArJ7VigB/oyrdnoebXGmDSjpS8
Lz119ixIRPA68LOvu3Ozd3iUz9K5B0ZnxJBEQWwCtDwhMisKg/AOnPu668xRhsRI
9C04KZh377DGBWQTvemzXxi+gU1qK5FVT9u6pbt+7majEoXNXpWPu65FoxIdfMNL
GKztL3avSaztbCu8MmKTXFje1z62mhWKKl0gs6e5nMVlUPMuczk9e/b30ZYXT+jl
R2FfWks6AgUeIK6mEkt3TcPK1EyuPY9m65d/aJynSPD2xt0/2f1d6eDvHH2Maa0i
+COPQw==
=n2Cw
-----END PGP PUBLIC KEY BLOCK-----
`

// verifyBinaryWithSignature verifies the binary file against its signature using the embedded public PGP key.
func verifyBinaryWithSignature(logs *logger.Logger, binaryPath string, signaturePath string) error {
	publicKey := publicKey
	keyRing, err := openpgp.ReadArmoredKeyRing(strings.NewReader(publicKey))
	if err != nil {
		logs.Error("failed to read public key: " + err.Error())
		return err
	}

	// Load the binary
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		logs.Error("failed to read binary file: " + err.Error())
		return err
	}

	// Load the signature
	signatureData, err := os.ReadFile(signaturePath)
	if err != nil {
		logs.Error("failed to read signature file: " + err.Error())
		return err
	}

	_, err = openpgp.CheckArmoredDetachedSignature(keyRing, bytes.NewReader(binaryData), bytes.NewReader(signatureData))
	if err != nil {
		logs.Error("signature verification failed: " + err.Error())
		return err
	}
	logs.Info("Signature verification successful for binary: " + binaryPath)
	return nil
}
