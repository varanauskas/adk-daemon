[![GoDoc](https://godoc.org/github.com/AidosKuneen/aidosd?status.svg)](https://godoc.org/github.com/AidosKuneen/aidosd)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/AidosKuneen/aidosd/LICENSE)
# adk-daemon: aidosd.v2

aidosd (v2) is a deamon which acts as bitcoind for adk. 
This version has been built specifically for network mesh version 2+

For now implemented APIs are:

## NOTE: DON'T USE MULTIPLE ACCOUNTS. Account feature will be removed in a later version.

* `getnewaddress`
* `listaccounts`
* `listaddressgroupings`
* `validateaddress`
* `settxfee`
* `walletpassphrase`
* `sendmany`
* `sendfrom`
* `gettransaction`
* `getbalance`
* `sendtoaddress`

and `walletnotify` feature.

Refer Bitcoin API Reference (e.g. [here](https://bitcoin.org/en/developer-reference#rpcs)) for more details
about how to call these APIs.

See [incompatibility lists](https://github.com/AidosKuneen/adk-daemon/blob/master/incompatibilities.md)
for details about incompatibilities with Bitcoin APIs.


# Reqirements

* go 1.15+
* gcc (for linux)
* mingw (for windows)

# Build

```
	$ mkdir go
	$ cd go
	$ mkdir src
	$ mkdir bin
	$ mkdir pkg
	$ exoprt GOPATH=`pwd`
	$ cd src
	$ go get -u github.com/AidosKuneen/adk-daemon
	$ cd github.com/AidosKuneen/adk-daemon
	$ go build -o aidosd
```

NOTE: if you dont specify "-o aidosd" durign the build, your executable will be named adk-daemon instead (still works..). But if you want to replace an old installation you should rename it to "aidosd".

# Configuration

Configurations are in `aidosd.conf`.

 * `rpcuser` : Username for JSON-RPC connections 
 * `rpcpassword`: Password for JSON-RPC connections 
 * `rpcport`: Listen for JSON-RPC connections on <port> (default: 8332) 
 * `walletnotify`: Execute command when a transaction comes into a wallet (%s in cmd is replaced by bundle ID) 
 * `aidos_node`: Host address of an Aidos node server , which must be configured for wallet.
 * `passphrase`: Set `false` if your program sends tokens withtout `walletpassphrase` (default :true) .
 * `tag`: Set your identifier. You can use charcters 9 and A~Z and don't use other ones, and it must be under 20 characters.

 This is used as tag in transactions aidosd sends.

Note that `aidosd` always encrypts seeds with AES regardless `passphrase` settings.

Examples of `aidosd.conf`:

```
rpcuser=put_your_username
rpcpassword=put_your_password
rpcport=8332
walletnotify=/home/adk/script/receive.sh %s
aidos_node=http://api1.mainnet.aidoskuneen.com:14266
testnet=false
passphrase=true
tag="AWESOME9EXCHANGER"
```

# Usage

CHANGES TO VERSION 1 (aidosd):

## Set up a new wallet

start aidosd with command
```
./aidosd -generate
```

This will set up a new wallet and generate a new seed for you. Make sure you write down the seed as it will only be shown once!

## Set up a wallet by importing an existing seed

start aidosd with command
```
./aidosd -import
```

This will set up a new wallet, but prompt you for a seed. Once entered it will scan the Mesh for existing transactions and balances, and it will pre-calculate 50000 addresses (please be patient, this can take a while)

## Set up a wallet by importing an existing seed from a V1 aidosd installation (aidosd.db)

In order to UPGRADE an existing aidosd setup (v1), simply overwrite the old v1 aidosd executable with the new v2 one.
Then simply start aidosd with command
```
./aidosd
```

aidosd v2 will automatically recognize that you don't have a v2 seed database, so it will prompt you interactively if you want to scan the old database (aidosd.db) for the seed, and for existing/known addresses. (Note, it will also ask you the same if you use the -import parameter, but it sees that there is an old v1 aidosd.db file)


This will scan and import known addresses from the old database, and scan the Mesh for existing transactions and balances, and it will pre-calculate 50000 addresses (please be patient, this can take a while)


## Once the wallet is set up:

In order to start aidosd, simply run it from commandline without parameter. It will prompt you for the password (unless you have set it via ENV parameter AIDOSD_PASSWORD )

```
$ ./aidosd
enter password: <input your password> 
```

Then `aidosd` starts to run in background.

```
$ ./aidosd
Enter password: 
starting the aidosd server at port http://0.0.0.0:8332
aidosd has started
```


To know if it is still running, run:

```
	$ ./aidosd status
```

This prints the status ("running" or "stopped").


When you want to stop:

```
	$ ./aidosd stop
```
