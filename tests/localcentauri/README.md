# Localcentauri

Localcentauri is a complete centauri testnet containerized with Docker and orchestrated with a simple docker-compose file. Localcentauri comes preconfigured with opinionated, sensible defaults for a standard testing environment.

Localcentauri comes in two flavors:

1. No initial state: brand new testnet with no initial state. 
2. With mainnet state: creates a testnet from a mainnet state export

## Prerequisites

Ensure you have docker and docker-compose installed:

## 1. Localcentauri - No Initial State

The following commands must be executed from the root folder of the centauri repository.

1. Make any change to the centauri code that you want to test

2. Initialize Localcentauri:

```bash
make localnet-init
```

The command:

- Builds a local docker image with the latest changes
- Cleans the `$HOME/.centauri` folder

3. Start Localcentauri:

```bash
make localnet-start
```

> Note
>
> You can also start Localcentauri in detach mode with:
>
> `make localnet-startd`
4. (optional) Add your validator wallet and 9 other preloaded wallets automatically:
```bash
make localnet-keys
```

- These keys are added to your `--keyring-backend test`
- If the keys are already on your keyring, you will get an `"Error: aborted"`
- Ensure you use the name of the account as listed in the table below, as well as ensure you append the `--keyring-backend test` to your txs
- Example: `centaurid tx bank send lo-test2 centauri1cyyzpxplxdzkeea7kwsydadg87357qnahakaks --keyring-backend test --chain-id Localcentauri`

5. You can stop chain, keeping the state with

```bash
make localnet-stop
```

6. When you are done you can clean up the environment with:

```bash
make localnet-clean
```

## 2. Localcentauri - With Mainnet State

Running Localcentauri with mainnet state is resource intensive and can take a bit of time.
It is recommended to only use this method if you are testing a new feature that must be thoroughly tested before pushing to production.

A few things to note before getting started. The below method will only work if you are using the same version as mainnet. In other words,
if mainnet is on v11.0.0 and you try to do this on a v12.0.0 tag or on main, you will run into an error when initializing the genesis.
(yes, it is possible to create a state exported testnet on a upcoming release, but that is out of the scope of this tutorial)


### Create a mainnet state export

1. Set up a node on mainnet (Example: http://cosmosia10.notional.ventures:11111/centauri/)

2. Take a state export snapshot with the following command:

```sh
cd $HOME
centaurid export > state_export.json
```

After a while (~15 minutes), this will create a file called `state_export.json` which is a snapshot of the current mainnet state.

### Use the state export in Localcentauri

1. Copy the `state_export.json` to the `localcentauri/state_export` folder within the centauri repo


2. Ensure you have docker and docker-compose installed


3. Build the `local:centauri` docker image:

```bash
make localnet-state-export-init
```

The command:

- Builds a local docker image with the latest changes
- Cleans the `$HOME/.centauri` folder

4. Start Localcentauri:

```bash
make localnet-state-export-start
```

> Note
>
> You can also start Localcentauri in detach mode with:
>
> `make localnet-state-export-startd`
When running this command for the first time, `local:centauri` will:
- Modify the provided `state_export.json` to create a new state suitable for a testnet
- Start the chain

You will then go through the genesis initialization process. This will take ~15 minutes.
You will then hit the first block (not block 1, but the block number after your snapshot was taken), and then you will just see a bunch of p2p error logs with some KV store logs.
**This will happen for about 1 hour**, and then you will finally hit blocks at a normal pace.


5. On your host machine, you can now query the state-exported testnet:

```sh
centaurid status
```

6. Here is an example command to ensure complete understanding:

```sh
centaurid tx bank send wallet centauri1jxa3ksucx7ter57xyuczvmk6qkeqmqvj37g237 100000ppica --chain-id localcentauri --keyring-backend test
```

7. You can stop chain, keeping the state with

```bash
make localnet-state-export-stop
```

8. When you are done you can clean up the environment with:

```bash
make localnet-state-export-clean
```