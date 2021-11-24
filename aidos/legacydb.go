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
   	"fmt"
    "github.com/AidosKuneen/gadk"
    "os"
    "time"
    "encoding/json"
    "github.com/boltdb-go/bolt"
   )
  var accountDB = []byte("accounts")
  type Account struct {
  	Name     string
  	Seed     gadk.Trytes `json:"-"`
  	EncSeed  []byte
  	Balances []Balance
  }

  type Balance struct {
  	gadk.Balance
  	Change int64
  }



  func listAddressesFromOldDB(db *bolt.DB) ([]gadk.Address){
    	  fmt.Println("Checking legacy db for accounts: ")
        retAddr := []gadk.Address{}
    		db.View(func(tx *bolt.Tx) error {
    			acc, err2 := listAccount(tx)
    			if err2 != nil {
    				return err2
    			}
    			var cnt int = 0
    			for idx, ac := range acc {
            time.Sleep( 2 * time.Second)
    				fmt.Printf("Account found: Account number %v : %s \n", idx, ac.Name)
    				cnt++
            for _, bal := range ac.Balances {
                addr := bal.Balance.Address
                retAddr = append(retAddr, addr)
            }

    			}
    			return nil
    		})
        fmt.Println("")
        fmt.Println("Imported",len(retAddr),"old known addresses from db.")
        fmt.Println("")
        time.Sleep( 2 * time.Second)
        return retAddr
  }

  func getSeedFromOldDB () (string, bool) {
        //
        db, errDB := bolt.Open("aidosd.db", 0600, nil)
        Pcheck(errDB)
        defer db.Close()
        
        fmt.Println("Checking legacy db for existing SEED: ")
        //
        // first check if we have the password ENV
        passwd := []byte(os.Getenv("AIDOSD_PASSWORD"))

        if (len(passwd) >= 6){
           fmt.Println("Test 1: Found password in ENV AIDOSD_PASSWORD, trying that first to extract seed...")
           time.Sleep( 1 * time.Second)
           if correctLegacyPassword(passwd, db){
               fmt.Println("valid password.")
           } else {
             fmt.Println("invalid password.")
             passwd =[]byte{}
           }
           time.Sleep( 1 * time.Second)
        }

        if (len(passwd) < 6 && len(Aconf.RPCPassword) >=6){
           fmt.Println("Test 2: Found password in RPCPassword, trying that to extract seed...")
           time.Sleep( 1 * time.Second)
           passwd = []byte(Aconf.RPCPassword)
           if correctLegacyPassword(passwd, db){
               fmt.Println("valid password.")
           } else {
             fmt.Println("invalid password.")
             passwd =[]byte{}
           }
           time.Sleep( 1 * time.Second)
        }

        if (len(passwd) < 6) {
          fmt.Println("Could not automatically extract password to decrypt old seed, so")
          fmt.Println("")
          fmt.Println("Please enter the OLD password used in the legacy database to continue:")
          passwd = GetPasswd()
          if correctLegacyPassword(passwd, db){
              fmt.Println("valid password.")
          } else {
            fmt.Println("invalid password.")
            return "",false
          }
        }

        seed := ""
        found := false
        cntaddrs := -1
        db.View(func(tx *bolt.Tx) error {
          acc, err2 := listAccount(tx)
          if err2 != nil {
            return nil
          }
          for idx, ac := range acc {
            fmt.Printf("Account found: Account number %v : %s \n", idx, ac.Name)
            //
            crypt, errC := newAESCrpto(passwd, db)
            if errC != nil {
              fmt.Println(errC)
              continue
            }
            var errT error
            ac.Seed, errT = gadk.ToTrytes(string(crypt.decrypt(ac.EncSeed)))
            if (errT != nil) {
              continue
            }
            if len(ac.Balances) > cntaddrs && len(ac.Seed)==81{
                cntaddrs = len(ac.Balances)
                seed = string(ac.Seed)
            }
          }
          return nil
        })
        if len(seed) == 81 {
          fmt.Println("Loaded Seed: ",seed[0:3]+"..."+seed[78:81])
          found = true
        } else {
            fmt.Println("No seed found in legacy db ")
            found = false
        }
        return seed, found
  }


  func listAccount(tx *bolt.Tx) ([]Account, error) {
  	var asc []Account
  	// Assume bucket exists and has keys
  	b := tx.Bucket(accountDB)
  	if b == nil {
  		return nil, nil
  	}
  	c := b.Cursor()
  	for k, v := c.First(); k != nil; k, v = c.Next() {
  		var ac Account
  		if err := json.Unmarshal(v, &ac); err != nil {
  			return nil, err
  		}
  		asc = append(asc, ac)
  	}
  	return asc, nil // return specific account slice
  }
