use ::serde::{Deserialize, Serialize};
use rocket::{
    form::Form, get, http::Status, launch, post, response::Redirect, routes, FromForm, State,
};
use rocket_dyn_templates::{context, Template};
use serde_json::json;

#[derive(Serialize, Deserialize, FromForm, Debug)]
struct VmRequest {
    pub ethz_email: String,
    pub external_email: String,
    pub os: String,
    pub ssh_keys: String,
    pub hostname: String,
    pub cores: u32,
    pub ram: u32,
    pub disk: u32,
    pub wishes: Option<String>,
    pub bot: bool,
    #[field(name = "h-captcha-response")]
    h_captcha_response: String,
}

#[derive(Deserialize, Serialize, Debug)]
struct HCaptcha {
    sitekey: String,
    secret: String,
}

#[derive(Deserialize, Serialize, Debug)]
struct Environment {
    hcaptcha: HCaptcha,
}

#[get("/apply")]
fn get_apply(env: &State<Environment>) -> Template {
    Template::render("apply", json!({"captcha_sitekey": &env.hcaptcha.sitekey}))
}

#[post("/apply", data = "<data>")]
async fn post_apply(data: Form<VmRequest>, env: &State<Environment>) -> Result<Redirect, Status> {
    let captcha = reqwest::Client::new()
        .post("https://hcaptcha.com/siteverify")
        .query(&[
            ("secret", &env.hcaptcha.secret),
            ("response", &data.h_captcha_response),
        ])
        .send()
        .await
        .map_err(|e| {
            log::warn!("error while verifying hcaptcha response: {e:?}");
            Status::InternalServerError
        })?;

    if !captcha.status().is_success() {
        log::warn!("captcha response failed verification: {captcha:?}");
        return Err(Status::Forbidden);
    }

    Ok(Redirect::to("/success"))
}

#[get("/success")]
fn get_success() -> Template {
    Template::render("success", context! {})
}

#[launch]
fn rocket() -> _ {
    dotenv::dotenv().ok();
    env_logger::init();

    let env: Environment = serde_env::from_env().expect("failed to deserialize environment");

    rocket::build()
        .attach(Template::fairing())
        .manage(env)
        .mount("/", routes![get_apply, post_apply, get_success])
}
