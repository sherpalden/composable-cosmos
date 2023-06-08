#!/bin/sh
# set -e 
# set -o pipefail

# Home path in the docker container
CENTAURI_HOME=/centauri/.centauri
CONFIG_FOLDER=$CENTAURI_HOME/config

# val - centauri1jxa3ksucx7ter57xyuczvmk6qkeqmqvj37g237
DEFAULT_MNEMONIC="blame tube add leopard fire next exercise evoke young team payment senior know estate mandate negative actual aware slab drive celery elevator burden utility"
DEFAULT_CHAIN_ID="localcentauri"
DEFAULT_MONIKER="val"

# Override default values with environment variables
MNEMONIC=${MNEMONIC:-$DEFAULT_MNEMONIC}
CHAIN_ID=${CHAIN_ID:-$DEFAULT_CHAIN_ID}
MONIKER=${MONIKER:-$DEFAULT_MONIKER}

install_prerequisites () {
    apk add -q --no-cache \
        dasel \
        python3 \
        py3-pip
}

edit_config () {
    # Remove seeds
    dasel put string -f $CONFIG_FOLDER/config.toml '.p2p.seeds' ''

    # Disable fast_sync
    dasel put bool -f $CONFIG_FOLDER/config.toml '.fast_sync' 'false'

    # Expose the rpc
    dasel put string -f $CONFIG_FOLDER/config.toml '.rpc.laddr' "tcp://0.0.0.0:26657"

    # minimum-gas-prices config in app.toml, empty string by default
    dasel put string -f $CONFIG_FOLDER/app.toml 'minimum-gas-prices' "0ppica"
}

if [[ ! -d $CONFIG_FOLDER ]]
then

    install_prerequisites

    echo "Chain ID: $CHAIN_ID"
    echo "Moniker:  $MONIKER"

    echo $MNEMONIC | centaurid init $MONIKER -o --chain-id=$CHAIN_ID --home $CENTAURI_HOME
    echo $MNEMONIC | centaurid keys add my-key --recover --keyring-backend test 2> /dev/null

    ACCOUNT_PUBKEY=$(centaurid keys show --keyring-backend test my-key --pubkey | dasel -r json '.key' --plain)
    ACCOUNT_ADDRESS=$(centaurid keys show -a --keyring-backend test my-key --bech acc)
    ACCOUNT_ADDRESS_JSON=$(centaurid keys show --keyring-backend test my-key --output json | dasel -r json '.pubkey' --plain)
    echo "Account pubkey:  $ACCOUNT_PUBKEY"
    echo "Account address: $ACCOUNT_ADDRESS"

    ACCOUNT_HEX_ADDRESS=$(centaurid debug pubkey $ACCOUNT_ADDRESS_JSON --home $CENTAURI_HOME | grep Address | cut -d " " -f 2)    
    ACCOUNT_OPERATOR_ADDRESS=$(centaurid debug addr $ACCOUNT_HEX_ADDRESS --home $CENTAURI_HOME | grep Val | cut -d " " -f 3)    

    VALIDATOR_PUBKEY_JSON=$(centaurid tendermint show-validator --home $CENTAURI_HOME)
    VALIDATOR_PUBKEY=$(echo $VALIDATOR_PUBKEY_JSON | dasel -r json '.key' --plain)
    VALIDATOR_HEX_ADDRESS=$(centaurid debug pubkey $VALIDATOR_PUBKEY_JSON --home $CENTAURI_HOME | grep Address | cut -d " " -f 2)    
    VALIDATOR_ACCOUNT_ADDRESS=$(centaurid debug addr $VALIDATOR_HEX_ADDRESS --home $CENTAURI_HOME | grep Acc | cut -d " " -f 3)
    VALIDATOR_OPERATOR_ADDRESS=$(centaurid debug addr $VALIDATOR_HEX_ADDRESS --home $CENTAURI_HOME | grep Val | cut -d " " -f 3)    
    # VALIDATOR_CONSENSUS_ADDRESS=$(centaurid debug bech32-convert $VALIDATOR_OPERATOR_ADDRESS -p centaurivalcons  --home $CENTAURI_HOME)   
    VALIDATOR_CONSENSUS_ADDRESS=$(centaurid tendermint show-address --home $CENTAURI_HOME)   
    echo "Validator pubkey:  $VALIDATOR_PUBKEY"
    echo "Validator address: $VALIDATOR_ACCOUNT_ADDRESS"
    echo "Validator operator address: $VALIDATOR_OPERATOR_ADDRESS" 
    echo "Validator consensus address: $VALIDATOR_CONSENSUS_ADDRESS"    

    python3 -u /centauri/testnetify.py \
        -i /centauri/state_export.json \
        --output $CONFIG_FOLDER/genesis.json \
        -c $CHAIN_ID \
        --validator-hex-address $VALIDATOR_HEX_ADDRESS \
        --validator-operator-address $ACCOUNT_OPERATOR_ADDRESS \
        --validator-consensus-address $VALIDATOR_CONSENSUS_ADDRESS \
        --validator-pubkey $VALIDATOR_PUBKEY \
        --account-pubkey $ACCOUNT_PUBKEY \
        --account-address $ACCOUNT_ADDRESS \
        --prune-ibc

    edit_config
fi

centaurid validate-genesis --home $CENTAURI_HOME
centaurid start --x-crisis-skip-assert-invariants --home $CENTAURI_HOME