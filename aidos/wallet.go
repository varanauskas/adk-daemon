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
//   	"flag"
   	"fmt"
    "log"
    "time"
    "bufio"
    "strconv"
    "github.com/AidosKuneen/gadk"
// //  	"log"
   	"os"
    "io"
    "strings"
    "github.com/boltdb-go/bolt"
    //     "io/ioutil"
    "sync"
   )

   var filemutex sync.Mutex

   var hardfilemutex_transactions_dat sync.Mutex
   var hardfilemutex_balances_dat sync.Mutex

   var AddressMap map[gadk.Address]int64
   var AddressMapByIndex map[int]gadk.Address
   var LastUsedAddressIndex int

  func OpenWallet(passwd []byte){
      fmt.Println()
      fmt.Println(" **** Opening wallet ****")
      LoadSeedFromFile(passwd)
  }

  const maxblocks = 10  // up to 50000 addresses...
  const blocksize = 5000

  func GetNewAddress() (gadk.Address){
      filemutex.Lock()
      LastUsedAddressIndex++;
      newAddressIdx := LastUsedAddressIndex
      if (len(AddressMapByIndex[newAddressIdx])) != 81 {
        filemutex.Unlock()
        log.Fatal("ERROR: NO ADDRESS FOUND AT INDEX ",newAddressIdx," Either address numbers for seed exhausted (index > ",(maxblocks*blocksize),") or balances.dat is corrupted.")
        os.Exit(1)
      }
      // save file
      fmt.Println("Address status has changed. Updating balance file balances.dat...")
      filemutex.Unlock()
      //StoreBalance has its own lock
      StoreBalances(AddressMap,AddressMapByIndex,LastUsedAddressIndex)

      return AddressMapByIndex[newAddressIdx]
  }

  func ScanStoreAddressesAndBalancesINIT(generatedNewSeed bool){
    if !generatedNewSeed {
        fmt.Println(" **** Scanning Mesh for Balances *****")
    } else {
        fmt.Println(" **** Generating Wallet Addresses (this can take a while, please wait) *****")
    }
    fmt.Println("");
    knownLegacyAddresses := []gadk.Address{}

    if !generatedNewSeed { // importing existing seed
          fmt.Println("Looking for existing aidosd.db to import known addresses")
          if _, err := os.Stat("aidosd.db"); err == nil {
              fmt.Println("")
              fmt.Print("--> Old AIDOSD.DB file exists. Do you want to import know addresses? ")
              if AskConfirm() {
                  fmt.Println("")
                  fmt.Println("User choice: IMPORTING addresses from aidosd.db. Please wait...")
                  fmt.Println("")
                  db, errDB := bolt.Open("aidosd.db", 0600, nil)
                  Pcheck(errDB)
                  defer db.Close()
                  knownLegacyAddresses = listAddressesFromOldDB(db)

              } else {
                fmt.Println("")
                fmt.Println("User choice: NOT IMPORTING addresses from aidosd.db")
                fmt.Println("")
                time.Sleep(1 * time.Second)
              }
          } else {
            fmt.Println("aidosd.db does not exist. skipping aidosd.db import.")
            time.Sleep(4 * time.Second)
            fmt.Println("")
          }
    }

    fmt.Println("Generating addresses from seed to scan... please wait")
    seedTrytes, errTrytes := gadk.ToTrytes(Seed)
    Pcheck(errTrytes)
    highestIndexWithBalance := -1
    highestIndexWithKnownLegacyAddress := -1

    addressMap := make(map[gadk.Address]int64)
    addressMapByIndex := make(map[int]gadk.Address)

    countLegacyFoundTOTAL := int64(0)

    for block := 0; block < maxblocks; block++{
        blockBalTotal := int64(0)
        countLegacyFound := int64(0)
        fmt.Println("checking addresses", block * blocksize,"to", block * blocksize+blocksize,"of",maxblocks*blocksize)
        addresses, _ := gadk.NewAddresses(seedTrytes, block * blocksize, blocksize, 2)

        if !generatedNewSeed {

                  balances, berr := Api.GetBalances(addresses,100)
                  if (berr!=nil){
                      fmt.Println("Error connecting to node. Rolling back seed import.")
                      os.Remove("seed.enc")
                  }
                  Pcheck(berr)
                  for idx, bal := range balances.Balances {
                     addressMapByIndex[(block * blocksize + idx)] = addresses[idx]
                     if bal > 0 {
                       highestIndexWithBalance =  block * blocksize + idx
                       blockBalTotal += bal
                     }
                     if AddressInList(addresses[idx], knownLegacyAddresses) {
                        highestIndexWithKnownLegacyAddress =  block * blocksize + idx
                        countLegacyFound ++
                        countLegacyFoundTOTAL ++
                     }

                     addressMap[addresses[idx]] = bal
                  }
                  fmt.Println(" ---> Total balances in block:",blockBalTotal, "( ~",blockBalTotal/100000000,"ADK )")
                  if (len(knownLegacyAddresses)>0){
                      fmt.Println(" ---> Legacy addresses found in block:",countLegacyFound)
                  }
        } else {// else setting up a new empty seed.
            for idx, _ := range addresses {
              addressMapByIndex[(block * blocksize + idx)] = addresses[idx]
              addressMap[addresses[idx]] = 0
              highestIndexWithBalance = -1
            }
        }
    }

    if (len(knownLegacyAddresses) > 0 && countLegacyFoundTOTAL == 0){
          fmt.Println(" ******************************************** ");
          fmt.Println(" ** WARNING: None of the addresses imported from the legacy database aidosd.db ");
          fmt.Println(" **          were found on the SEED you provided.  ");
          fmt.Println(" **           ");
          fmt.Println(" **          Please check that you are using the same SEED as");
          fmt.Println(" **          used for the old legacy database aidosd.db.");
          fmt.Println(" ******************************************** ");
          log.Fatal("seed missmatch old aidosd.db database and new seed (seed.enc)")
    }

    fmt.Println(" ");
    if !generatedNewSeed { // importing existing seed
        fmt.Println("The last known address found (with balance, or imported) is: ");
    } else {
        fmt.Println("First address in wallet is: ");
    }

    lastKnownAddressIdx := highestIndexWithBalance
    if highestIndexWithKnownLegacyAddress > lastKnownAddressIdx {
      lastKnownAddressIdx = highestIndexWithKnownLegacyAddress
    }
    if lastKnownAddressIdx < 0 { // if lastKnownAddressIdx is still -1 here, then this is a brand new seed
      lastKnownAddressIdx = 0 // set first address to be the last one...
    }
    fmt.Println(addressMapByIndex[lastKnownAddressIdx],"at seed index",lastKnownAddressIdx)
    StoreBalances(addressMap,addressMapByIndex,lastKnownAddressIdx)

  }


  func ScanAddressesForBalanceChanges(scanAdditional int){

        balancesUpdated := false

        for block := 0; block < maxblocks; block++{
            addresses := []gadk.Address{}
            for index := block * blocksize; index < block * blocksize + blocksize &&
                                            index < LastUsedAddressIndex + scanAdditional; index ++ {
              addresses = append(addresses, AddressMapByIndex[index] )
            }
            highestToScan := block * blocksize+blocksize;
            if highestToScan > LastUsedAddressIndex+scanAdditional {
              highestToScan = LastUsedAddressIndex+scanAdditional
            }
            if block * blocksize > LastUsedAddressIndex+scanAdditional {
              continue // skipping we're too high now
            }
            fmt.Println("checking addresses", block * blocksize,"to", highestToScan,"of",LastUsedAddressIndex+scanAdditional)

            if len(addresses) == 0 {
              break
            }
            balances, berr := Api.GetBalances(addresses,100)
            if berr != nil {
              fmt.Println("Error during Api.GetBalances... trying again later. ",berr, len(addresses))
              return
            }

            for idx, newBalance := range balances.Balances {
              oldBalance := AddressMap[addresses[idx]]
              if (oldBalance != newBalance) { // address balance has changed!
                fmt.Println("ADDRESS BALANCE UPDATE: ",addresses[idx])
                fmt.Println("  updated from",oldBalance,"to",newBalance)
                balancesUpdated = true
                AddressMap[addresses[idx]] = newBalance
                if ( block * blocksize + idx > LastUsedAddressIndex){
                  fmt.Println(" ---> AND: This is a new address, so increasing LastUsedAddressIndex from",LastUsedAddressIndex,"to",block * blocksize + idx)
                  LastUsedAddressIndex = block * blocksize + idx
                }
              }
            }
        }

        if balancesUpdated {
          fmt.Println("Address balances have changed. Updating balance file balances.dat...")
          StoreBalances(AddressMap,AddressMapByIndex,LastUsedAddressIndex)
          fmt.Println("balances.dat file written.")

        }
      }


  func ReadAddressesFromFile() (addressBalances map[gadk.Address]int64,  addressListByIndex map[int]gadk.Address, lastKnownAddressIndex int) {
    filemutex.Lock()
    defer filemutex.Unlock()

      hardfilemutex_balances_dat.Lock()
      file, err := os.Open("balances.dat")
      addressMap := make(map[gadk.Address]int64)
      addressMapByIndex := make(map[int]gadk.Address)
       if err != nil {
           hardfilemutex_balances_dat.Unlock()
           return addressMap, addressMapByIndex, -1
       }
       scanner := bufio.NewScanner(file)
       lastKnown := -1
       for scanner.Scan() {
           line := strings.TrimSuffix(scanner.Text(), "\n")
           fields := strings.Split(line, ",")
           if (len(fields)==4){
             lidx, err1 := strconv.Atoi(fields[0])
             bal, err2 := strconv.ParseInt(fields[3], 10, 64)
             addr, err3 := gadk.ToAddress(fields[2])
             if (err1 != nil || err2 != nil || err3 != nil){
               fmt.Println("ERROR in file balance dat. Invalid line format: ",line)
             }
             addressMap[addr] = bal
             addressMapByIndex[lidx] = addr
             if (fields[1]=="USED") {
                if lidx > lastKnown {
                  lastKnown = lidx
                }
             }
           }
       }
       file.Close();
       hardfilemutex_balances_dat.Unlock()
       LastUsedAddressIndex = lastKnown
       AddressMap = addressMap
       AddressMapByIndex = addressMapByIndex
       return addressMap, addressMapByIndex, LastUsedAddressIndex
  }

  func StoreBalances(addressBalances map[gadk.Address]int64,  addressListByIndex map[int]gadk.Address, lastKnownAddressIndex int){
      filemutex.Lock()
      defer filemutex.Unlock()

      fmt.Println("Storing balances in balances.dat");
      hardfilemutex_balances_dat.Lock()
      f, err := os.OpenFile("balances.dat", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
      if err != nil {
          log.Fatal(err)
      }
      w := bufio.NewWriter(f)
      // store list of accounts, by index and balance
      for idx := 0; idx < len(addressListByIndex); idx++{
        // write an address
        addrstatus := "NEW"
        if idx <= lastKnownAddressIndex {
          addrstatus = "USED"
        }
        line := strconv.Itoa(idx)+","+addrstatus+","+string(addressListByIndex[idx])+","+strconv.FormatInt(addressBalances[addressListByIndex[idx]],10)+"\n"
        if _, err := w.Write([]byte(line)); err != nil {
          panic(err)
        }
        if err = w.Flush(); err != nil {
          panic(err)
        }
      }

      if err := f.Close(); err != nil {
          log.Fatal(err)
      }
      hardfilemutex_balances_dat.Unlock()
      MakeBackup("balances.dat");
  }

  func StoreConfirmedTransactions(addressBalances map[gadk.Address]int64,  addressListByIndex map[int]gadk.Address) ([]gadk.Trytes){

      filemutex.Lock()
      defer filemutex.Unlock()

      fmt.Println("Storing confirmed transactions in transactions.dat");
      // store list of accounts, by index and balance
      addrWithBals := []gadk.Address{}

      for idx := 0; idx < len(addressListByIndex); idx++{
        //
        if addressBalances[addressListByIndex[idx]] > 0 {
           addrWithBals = append(addrWithBals, addressListByIndex[idx])
        }
        //
      }
      fmt.Println("Checking transactions for",len(addrWithBals),"addresses")

      ft := gadk.FindTransactionsRequest{
        Addresses: addrWithBals,
      }
      newTransactions := []gadk.Trytes{}

      tresp, err2 := Api.FindTransactions(&ft)
      if err2 != nil {
        fmt.Println("error connectiong to node. trying again later. ",err2)
        return newTransactions
      }

      fmt.Println("Found",len(tresp.Hashes),"transactions.")
      fmt.Println("Storing transactions in transactions.dat")

      hardfilemutex_transactions_dat.Lock()
      knownTransactions := ReadFileIntoSlice("transactions.dat")  // first load any if there is already...
      hardfilemutex_transactions_dat.Unlock()

      transMap := make(map[gadk.Trytes]int)
      for _, tr := range knownTransactions {
        trytes2, _:=gadk.ToTrytes(tr)
        transMap[trytes2] = 1
      }

      existing := len(transMap)
      fmt.Println(existing,"existing transactions")


      for _, trans := range tresp.Hashes {
          if transMap[trans] != 1 {
            newTransactions = append(newTransactions, trans)
          }
          transMap[trans] = 1
      }
      newTransactionCount := len(transMap)-existing
      fmt.Println("Added",newTransactionCount,"new transactions")

      if (newTransactionCount > 0){
            fmt.Println("Saving new transactions to transaction.dat")
            hardfilemutex_transactions_dat.Lock()
            f, err := os.OpenFile("transactions.dat", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644) // APPEND
            if err != nil {
                hardfilemutex_transactions_dat.Unlock()
                log.Fatal(err)
            }
            w := bufio.NewWriter(f)

            for _, trans := range newTransactions {
                line := string(trans)+"\n"
                if _, err := w.Write([]byte(line)); err != nil {
                  panic(err)
                }
                if err = w.Flush(); err != nil {
                  panic(err)
                }
            }

            if err := f.Close(); err != nil {
                log.Fatal(err)
            }
            hardfilemutex_transactions_dat.Unlock()
      }

      MakeBackup("transactions.dat");

      return newTransactions
  }

  func ReadFileIntoSlice(fpath string) ([]string){
        lines := []string{}
        file, err := os.Open(fpath)
         if err != nil {
             return lines
         }
         scanner := bufio.NewScanner(file)
         for scanner.Scan() {
             line := strings.TrimSuffix(scanner.Text(), "\n")
             if len(line) == 81 {
               lines = append(lines, line)
             }
         }
         file.Close();
         return lines
  }

   func AddressInList(a gadk.Address, list []gadk.Address) bool {
      for _, b := range list {
          if b == a {
              return true
          }
      }
      return false
  }

  func GetIndexForAddress(a gadk.Address) int {
     for idx, b := range AddressMapByIndex {
         if b == a {
            return idx
         }
     }
     return -1
  }

   var copymutex sync.Mutex
    func MakeBackup(filename string){
        if (filename == "transactions.dat") {
            hardfilemutex_transactions_dat.Lock()
            copy("transactions.dat","transactions.dat.bak")
            hardfilemutex_transactions_dat.Unlock()
        } else if (filename == "balances.dat") {
            hardfilemutex_balances_dat.Lock()
            copy("balances.dat","balances.dat.bak")
            hardfilemutex_balances_dat.Unlock()
        } else {
           copymutex.Lock()
           copy(filename,filename+".bak")
           copymutex.Unlock()
        }
    }

    func copy(src, dst string) (int64, error) {
          sourceFileStat, err := os.Stat(src)
          if err != nil {
                  return 0, err
          }
          if !sourceFileStat.Mode().IsRegular() {
                  return 0, fmt.Errorf("%s is not a regular file", src)
          }
          source, err := os.Open(src)
          if err != nil {
                  return 0, err
          }
          defer source.Close()
          destination, err := os.Create(dst)
          if err != nil {
                  return 0, err
          }
          defer destination.Close()
          nBytes, err := io.Copy(destination, source)
          return nBytes, err
  }
