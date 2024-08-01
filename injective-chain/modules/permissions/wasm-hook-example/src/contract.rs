#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    ensure_eq, to_binary, Addr, Binary, CosmosMsg, Deps, DepsMut, Env, MessageInfo, Response,
    StdError, StdResult,
};
use injective_cosmwasm::InjectiveMsgWrapper;

use cw2::set_contract_version;
use cw_storage_plus::Item;

use crate::error::ContractError;
use crate::msg::{ExecuteMsg, InstantiateMsg, MigrateMsg, QueryMsg};

pub(crate) const CONTRACT_NAME: &str = "wasm-hook-example";
pub(crate) const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

/// The owner of the contract. Typically a DAO. The contract owner may
/// unilaterally execute messages on this contract.
pub const OWNER: Item<Addr> = Item::new("owner");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    set_contract_version(
        deps.storage,
        format!("crates.io:{CONTRACT_NAME}"),
        CONTRACT_VERSION,
    )?;

    let owner = deps.api.addr_validate(&msg.owner)?;
    OWNER.save(deps.storage, &owner)?;

    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("owner", owner.into_string())
        .add_attribute("creator", info.sender))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response<InjectiveMsgWrapper>, ContractError> {
    match msg {
        ExecuteMsg::UpdateOwner { new_owner } => update_owner(deps, env, info.sender, new_owner),
        ExecuteMsg::ExecuteMsgs { msgs } => exec_msgs(deps.as_ref(), info.sender, msgs),
    }
}

pub fn update_owner(
    deps: DepsMut,
    _env: Env,
    sender: Addr,
    new_owner: String,
) -> Result<Response<InjectiveMsgWrapper>, ContractError> {
    let current_owner = OWNER.load(deps.storage)?;
    ensure_eq!(sender, current_owner, ContractError::Unauthorized {});

    let new_owner = deps.api.addr_validate(&new_owner)?;
    OWNER.save(deps.storage, &new_owner)?;

    Ok(Response::default()
        .add_attribute("action", "update_owner")
        .add_attribute("owner", new_owner.into_string()))
}

pub fn exec_msgs(
    deps: Deps,
    sender: Addr,
    msgs: Vec<CosmosMsg<InjectiveMsgWrapper>>,
) -> Result<Response<InjectiveMsgWrapper>, ContractError> {
    let current_owner = OWNER.load(deps.storage)?;
    ensure_eq!(sender, current_owner, ContractError::Unauthorized {});

    Ok(Response::default()
        .add_attribute("action", "execute_msgs")
        .add_messages(msgs))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    let owner = OWNER.load(deps.storage)?;
    match msg {
        QueryMsg::Owner {} => to_binary(&owner),
        QueryMsg::SendRestriction {
            from_address,
            to_address,
            action,
            amounts,
        } => {
            deps.api.debug(&format!(
                "{:?}: {:?} -> {:?} = {:?}",
                action, from_address, to_address, amounts,
            ));
            match to_address.as_str().to_lowercase().chars().last().unwrap() {
                '0'..='c' => Err(StdError::GenericErr {
                    // for to_address that ends with '0' .. 'c' chars return error
                    msg: format!(
                        "action {:?} is restricted for user {:?}",
                        action, to_address
                    )
                    .to_string(),
                }),
                'd' => loop {
                    // for to_address that ends with 'd' invoke out of gas panic
                },
                _ => to_binary(&env.contract.address), // any other is successfull replace to_addr with contract_addr
            }
        }
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(deps: DepsMut, _env: Env, _msg: MigrateMsg) -> Result<Response, ContractError> {
    // Set contract to version to latest
    set_contract_version(
        deps.storage,
        format!("crates.io:{CONTRACT_NAME}"),
        CONTRACT_VERSION,
    )?;
    Ok(Response::default())
}
