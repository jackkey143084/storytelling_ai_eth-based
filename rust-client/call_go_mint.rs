use anyhow::Result;
use serde::Serialize;
use reqwest::blocking::Client;

#[derive(Serialize)]
struct MintReq<'a> {
    to_address: &'a str,
    ipfs_cid: &'a str,
    checkpoint: &'a str,
}

pub fn call_mint(api_url: &str, to: &str, cid: &str, checkpoint_hex: &str) -> Result<String> {
    let client = Client::new();
    let req = MintReq { to_address: to, ipfs_cid: cid, checkpoint: checkpoint_hex };
    let res = client.post(&format!("{}/mint", api_url)).json(&req).send()?;
    let json: serde_json::Value = res.json()?;
    Ok(json["tx_hash"].as_str().unwrap_or_default().to_string())
}
