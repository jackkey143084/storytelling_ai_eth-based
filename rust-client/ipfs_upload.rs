use anyhow::Result;
use reqwest::blocking::Client;
use std::fs;

pub fn upload_text_to_ipfs(text: &str) -> Result<String> {
    let client = Client::new();
    let part = reqwest::blocking::multipart::Part::text(text.to_string());
    let form = reqwest::blocking::multipart::Form::new().part("file", part);
    let res = client.post("http://localhost:5001/api/v0/add").multipart(form).send()?;
    let j: serde_json::Value = res.json()?;
    let hash = j["Hash"].as_str().ok_or(anyhow::anyhow!("no hash"))?;
    Ok(hash.to_string())
}
