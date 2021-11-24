  // Copyright (c) 2017 Aidos Developer

  // Permission is hereby granted, free of charge, to any person obtaining a copy
  // of this software and associated documentation files (the "Software"), to deal
  // in the Software without restriction, including without limitation the rights
  // to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
  // copies of the Software, and to permit persons to whom the Software is
  // furnished to do so, subject to the following conditions:

  // The above copyright notice and this permission notice shall be included in
  // all copies or substantial portions of the Software.

  // THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
  // IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
  // FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
  // AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
  // LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
  // OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
  // THE SOFTWARE.

  package aidos

   import (
   "crypto/rand"
    "fmt"
    "golang.org/x/term"
    "syscall"
    "math/big"
    "crypto/aes"
    "crypto/cipher"
    "os"
    "bufio"
    "io/ioutil"
    "strings"
    "golang.org/x/crypto/scrypt"
    "github.com/AidosKuneen/gadk"
    b64 "encoding/base64"
   )

  var Seed string
  var SeedEncrypted []byte
  var Password []byte

  func ImportSeed(importseed string) {
        if len(importseed) == 81 {
            Seed = importseed
        } else {
          fmt.Println("Importing existing seed and restoring wallet from seed.")
          fmt.Println("Please enter the seed you would like to import (81 char, A-Z9 )")
          Seed = ReadSeedUser()
        }
        GetPwdAndStoreSeed()
  }

  func ReadSeedUser()(string){
    fmt.Print("-> ")
    reader := bufio.NewReader(os.Stdin)
    Seed, _ = reader.ReadString('\n')
    // convert CRLF to LF
    Seed = strings.Replace(Seed, "\n", "", -1)
    Seed = strings.Replace(Seed, "\r", "", -1)
    _, err := gadk.ToTrytes(Seed)
    if err != nil || len(Seed) != 81 {
      fmt.Println("Invalid seed. Seed has to be 81 characters long, and only consist of letters A-Z and the number 9")
      fmt.Println("")
      Seed = ReadSeedUser()
    }
    return Seed
  }

  func GenerateSeed() {

      fmt.Println("aidosd has generated your new SEED:")
      fmt.Println("")
      var err error
      Seed, err = GenerateRandomSEED(81)
      Pcheck(err)
      fmt.Println(Seed)
      fmt.Println("")
      fmt.Println("Please save your seed in a safe place. It will only be shown this once and stored in encrypted form.")
      fmt.Println("")
      GetPwdAndStoreSeed()
      //SeedDecrypted, _ := Decrypt(Password,SeedEncrypted)
      //fmt.Println(string(SeedDecrypted))
  }

  func LoadSeedFromFile(passwd []byte){
    fmt.Println("Loading encrypted seed from file.")
    if len(passwd) >= 6 {
      Password = passwd  // password provided
    } else {
      Password = GetPasswd()
    }
    //
    content, err := ioutil.ReadFile("seed.enc")
    Pcheck(err)
    sDecContent, err64 := b64.StdEncoding.DecodeString(string(content))
    Pcheck(err64)

    seedDecrypted, err2 := Decrypt(Password,sDecContent)
    seedTrytes, errTrytes := gadk.ToTrytes(string(seedDecrypted))

    if err2 != nil || errTrytes != nil || len(seedTrytes) != 81 {
        if len(passwd) >= 6 {
            fmt.Println("Incorrect Password provided via ENV. Please try again manually")
        } else {
            fmt.Println("Incorrect Password. Please try again.")
        }
        LoadSeedFromFile(nil)
    } else {
      Seed = string(seedDecrypted)
    }
    //fmt.Println(Seed)
  }

  func LoadSeedAPIUnlock(passwd []byte)(bool){
    fmt.Println("Loading encrypted seed from file (API unlock).")
    if len(passwd) >= 6 {
      Password = passwd  // password provided
    } else {
       return false
    }
    //
    content, err := ioutil.ReadFile("seed.enc")
    Pcheck(err)
    sDecContent, err64 := b64.StdEncoding.DecodeString(string(content))
    Pcheck(err64)

    seedDecrypted, err2 := Decrypt(Password,sDecContent)
    seedTrytes, errTrytes := gadk.ToTrytes(string(seedDecrypted))

    if err2 != nil || errTrytes != nil || len(seedTrytes) != 81 {
      return false
    } else {
      Seed = string(seedDecrypted)
    }
    return true
    //fmt.Println(Seed)
  }

  func GetPwdAndStoreSeed() {
    fmt.Println("Please now enter a password that will encrypt your seed locally and be used to")
    fmt.Println("operate aidosd, perform transactions, etc.")
    fmt.Println("")
    Password = GetPasswd()
    fmt.Print("Password set. Encrypting seed...")
    var err error
    SeedEncrypted, err = Encrypt(Password,[]byte(Seed))
    Pcheck(err)
    fmt.Println("done.")
    fmt.Print("Storing encrypoted seed in seed.enc...")
    f, err := os.Create("seed.enc")
    Pcheck(err)
    _, err = f.WriteString(b64.StdEncoding.EncodeToString(SeedEncrypted))
    Pcheck(err)
    f.Close()
    MakeBackup("seed.enc");
    fmt.Println("done")
  }

  func GetPasswd() []byte {
  	fmt.Print("Enter password: ")
  	pwd, err := term.ReadPassword(int(syscall.Stdin)) //int conversion is needed for win
  	fmt.Println("")
  	Pcheck(err)
    if len(pwd) < 6 {
      fmt.Println("Password too short.")
      fmt.Println("")
      pwd = GetPasswd()
    }
    return pwd
  }

  func GenerateRandomSEED(n int) (string, error) {
    const letters = "9ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    ret := make([]byte, n)
    for i := 0; i < n; i++ {
      num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
      if err != nil {
        return "", err
      }
      ret[i] = letters[num.Int64()]
    }

    return string(ret), nil
  }


  func Encrypt(key, data []byte) ([]byte, error) {
    key, salt, err := DeriveKey(key, nil)
    if err != nil {
        return nil, err
    }

    blockCipher, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(blockCipher)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err = rand.Read(nonce); err != nil {
        return nil, err
    }

    ciphertext := gcm.Seal(nonce, nonce, data, nil)

    ciphertext = append(ciphertext, salt...)

    return ciphertext, nil
}

func Decrypt(key, data []byte) ([]byte, error) {
    salt, data := data[len(data)-32:], data[:len(data)-32]

    key, _, err := DeriveKey(key, salt)
    if err != nil {
        return nil, err
    }

    blockCipher, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(blockCipher)
    if err != nil {
        return nil, err
    }

    nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]

    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, err
    }

    return plaintext, nil
}

func DeriveKey(password, salt []byte) ([]byte, []byte, error) {
    if salt == nil {
        salt = make([]byte, 32)
        if _, err := rand.Read(salt); err != nil {
            return nil, nil, err
        }
    }

    key, err := scrypt.Key(password, salt, 1048576, 8, 1, 32)
    if err != nil {
        return nil, nil, err
    }

    return key, salt, nil
}
