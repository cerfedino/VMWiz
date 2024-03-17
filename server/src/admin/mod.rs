use crate::error::Res;
use crate::Environment;
use anyhow::Context;
use rocket::{get, routes, Build, Rocket, State};
use rocket_dyn_templates::{context, Template};

#[get("/free-ips")]
async fn free_ips(env: &State<Environment>) -> Res<Template> {
    // TODO do not hardcode subnet
    let ips: Vec<String> = crate::netcenter::api::free_ipv4(&env.netcenter, "192.33.91.0")
        .await
        .context("failed to fetch IPs from netcenter")?
        .into_iter()
        .map(|ip| ip.to_string())
        .collect();
    Ok(Template::render("ips", context! {ips}))
}

pub fn mount(rkt: Rocket<Build>, env: &Environment) -> Rocket<Build> {
    if env.deployment.is_prod() {
        panic!("THIS IS A WORK IN PROGRESS, DO NOT USE IN PROD.");
    }

    rkt.mount("/admin", routes![free_ips])
}
