use crate::NetCenter;
use serde::{Deserialize, Serialize};
use std::net::IpAddr;

#[derive(Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
#[serde(rename = "freeIp")]
struct FreeIp {
    ip: IpAddr,
    ip_subnet: IpAddr,
    ip_mask: u8,
    subnet_and_mask: String,
    subnet_name: String,
}

#[derive(Serialize, Deserialize)]
struct FreeIps {
    #[serde(rename = "$value")]
    entries: Vec<FreeIp>,
}

fn client() -> reqwest::Client {
    reqwest::ClientBuilder::new()
        .https_only(true)
        .user_agent("SOSETH netcenter automation (vsos-support@sos.ethz.ch)")
        .build()
        .unwrap()
}

pub async fn free_ipv4(cfg: &NetCenter, subnet: &str) -> anyhow::Result<Vec<IpAddr>> {
    let url = format!("https://www.netcenter.ethz.ch/netcenter/rest/nameToIP/freeIps/v4/{subnet}");

    let res = client()
        .get(url)
        .basic_auth(&cfg.user, Some(&cfg.pass))
        .send()
        .await?;

    let res = res.text().await?;
    let ips: FreeIps = serde_xml_rs::from_str(&res)?;
    let ips = ips.entries.into_iter().map(|entry| entry.ip).collect();

    Ok(ips)
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct NameToIp {
    ip: IpAddr,
    ip_subnet: IpAddr,
    fqname: String,
    // forward
    // reverse
    ttl: u32,
    // dhcp
    // ddns
    isg_group: String,
    // views
}

//<usedIps>
//<usedIp>
//<ip>192.33.91.50</ip>
//<ipSubnet>192.33.91.0</ipSubnet>
//<fqname>app.vsos.ethz.ch</fqname>
//<forward>Y</forward>
//<reverse>Y</reverse>
//<ttl>3600</ttl>
//<dhcp>N</dhcp>
//<ddns>N</ddns>
//<isgGroup>adm-soseth</isgGroup>
//<lastDetection>2024-03-19 12:33</lastDetection>
//<views>
//<view>extern</view>
//<view>intern</view>
//</views>
//</usedIp>
//</usedIps>

// known endpoints
// GET /netcenter/rest/nameToIP/freeIps/v4/{subnet}
// GET /netcenter/rest/nameToIP/usedIps/v4/{subnet}
// GET /netcenter/rest/nameToIP/{ip}
// GET /netcenter/rest/nameToIP/{ip}/{hostname}
// GET /netcenter/rest/alias/{alias}
// GET /netcenter/rest/alias/hostName/{hostname}
// GET /netcenter/rest/user/listAllIsgs
// GET /netcenter/rest/user/listIsgByUser
