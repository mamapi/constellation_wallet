package main

import (
	"os"

	"golang.org/x/crypto/bcrypt"
)

func (a *WalletApplication) LoginError(errMsg string) {
	if errMsg != "" {
		a.RT.Events.Emit("login_error", errMsg, true)
	}
}

func (a *WalletApplication) Login(keystorePath, keystorePassword, keyPassword, alias string) bool {

	os.Setenv("CL_STOREPASS", keystorePassword)
	os.Setenv("CL_KEYPASS", keyPassword)

	a.log.Warnln("Login Before: ", a.wallet.WalletAlias, alias)

	a.wallet.WalletAlias = alias

	a.log.Warnln("Login After: ", a.wallet.WalletAlias, alias)

	if err := a.DB.First(&a.wallet, "wallet_alias = ?", alias).Error; err != nil {
		a.log.Errorf("Unable to query database object for new wallet. Reason: ", err)
		a.LoginError("Access Denied. Alias not found.")
		return false
	}

	a.log.Warnln("Login AfterAFTER: ", a.wallet.WalletAlias, alias)

	if a.WalletKeystoreAccess() {
		a.KeyStoreAccess = true
	} else {
		a.KeyStoreAccess = false
		a.LoginError("Access Denied. Please make sure that you have typed in the correct credentials.")
		return false
	}

	if keystorePath == "" {
		a.log.Warnln("No path to keystore provided")
	}

	a.DB.Model(&a.wallet).Update("KeystorePath", keystorePath)
	a.log.Infoln("PrivateKey path: ", keystorePath)

	// Check password strings against salted hashes stored in DB. Also make sure KeyStore has been accessed.
	if a.CheckAccess(keystorePassword, a.wallet.KeystorePasswordHash) && a.CheckAccess(keyPassword, a.wallet.KeyPasswordHash) && a.KeyStoreAccess {
		a.UserLoggedIn = true

		// os.Setenv("CL_STOREPASS", keystorePassword)
		// os.Setenv("CL_KEYPASS", keyPassword)

	} else {
		a.UserLoggedIn = false
		a.LoginError("Access Denied. Please make sure that you have typed in the correct credentials.")
	}

	if a.UserLoggedIn && a.KeyStoreAccess && !a.NewUser {
		a.initExistingWallet(keystorePath)
	}

	a.NewUser = false

	return a.UserLoggedIn
}

func (a *WalletApplication) LogOut() {
	a.UserLoggedIn = true
}

func (a *WalletApplication) ImportKey() string {
	a.paths.EncPrivKeyFile = a.RT.Dialog.SelectFile()
	if a.paths.EncPrivKeyFile == "" {
		a.LoginError("Access Denied. No key path detected.")
	}

	a.log.Info("Path to imported key: " + a.paths.EncPrivKeyFile)
	return a.paths.EncPrivKeyFile
}

func (a *WalletApplication) SelectDirToStoreKey() string {
	a.paths.EncPrivKeyFile = a.RT.Dialog.SelectSaveFile()
	if a.paths.EncPrivKeyFile == "" {
		a.LoginError("No directory detected. Please try again.")
	}
	a.log.Info(a.paths.EncPrivKeyFile[len(a.paths.EncPrivKeyFile)-4:])
	if a.paths.EncPrivKeyFile[len(a.paths.EncPrivKeyFile)-4:] != ".p12" {
		a.paths.EncPrivKeyFile = a.paths.EncPrivKeyFile + ".p12"
	}
	return a.paths.EncPrivKeyFile
}

func (a *WalletApplication) GenerateSaltedHash(s string) (string, error) {
	saltedBytes := []byte(s)
	hashedBytes, err := bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	hash := string(hashedBytes[:])
	return hash, nil
}

func (a *WalletApplication) CheckAccess(password, passwordHash string) bool {
	err := a.Compare(password, passwordHash)
	if err != nil {
		a.log.Warnln("User tried to login with the wrong credentials!")
		return false
	} else {
		a.log.Infoln("Password check OK")
	}
	return true
}

func (a *WalletApplication) Compare(s, hash string) error {
	incoming := []byte(s)
	existing := []byte(hash)
	return bcrypt.CompareHashAndPassword(existing, incoming)
}