package crypto

import (
	"encoding/base64"
	"gateway/crypto"
	"gateway/logreport"

	corevm "gateway/core/vm"

	"github.com/robertkrimen/otto"
)

// IncludeEncryption adds the AP.Crypto.encrypt and AP.Crypto.decrypt functions in
// the supplied Otto VM.
func IncludeEncryption(vm *otto.Otto, accountID int64, keySource corevm.DataSource) {
	setEncrypt(vm, accountID, keySource)
	setDecrypt(vm, accountID, keySource)

	scripts := []string{
		// Ensure the top level AP object exists or create it
		"var AP = AP || {};",
		// Create the Crypto object
		"AP.Crypto = AP.Crypto || {};",
		"AP.Crypto.encrypt = _encrypt; delete _encrypt;",
		"AP.Crypto.decrypt = _decrypt; delete _decrypt;",
	}

	for _, s := range scripts {
		vm.Run(s)
	}
}

func setEncrypt(vm *otto.Otto, accountID int64, keySource corevm.DataSource) {
	vm.Set("_encrypt", func(call otto.FunctionCall) otto.Value {
		data, err := getData(call)
		if err != nil {
			logreport.Println(err)
			return undefined
		}

		o, err := corevm.GetArgument(call, 1)
		if err != nil {
			logreport.Println(err)
			return undefined
		}

		key, algorithm, tag, err := getOptions(o, keySource, accountID)
		if err != nil {
			logreport.Println(err)
			return undefined
		}

		result, err := crypto.Encrypt([]byte(data), key, algorithm, tag)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		val, err := vm.ToValue(result)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		return val
	})
}

func setDecrypt(vm *otto.Otto, accountID int64, keySource corevm.DataSource) {
	vm.Set("_decrypt", func(call otto.FunctionCall) otto.Value {
		ds, err := getData(call)
		if err != nil {
			logreport.Println(err)
			return undefined
		}

		o, err := corevm.GetArgument(call, 1)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		key, algorithm, tag, err := getOptions(o, keySource, accountID)
		if err != nil {
			logreport.Println(err)
			return undefined
		}

		data, err := base64.StdEncoding.DecodeString(ds)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		result, err := crypto.Decrypt(data, key, algorithm, tag)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		val, err := vm.ToValue(result)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		return val
	})
}
