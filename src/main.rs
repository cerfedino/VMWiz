use std::env;

use ::serde::{Deserialize, Serialize};
use rocket::{form::Form, response::Redirect, *};
use rocket_dyn_templates::{context, Template};

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

struct HCaptchaParams {
    sitekey: String,
    secret: String,
}

impl HCaptchaParams {
    fn from_env() -> Self {
        Self {
            sitekey: env::var("HCAPTCHA_SITEKEY").unwrap(),
            secret: env::var("HCAPTCHA_SECRET").unwrap(),
        }
    }
}

#[get("/apply")]
fn get_apply(hc: &State<HCaptchaParams>) -> Template {
    Template::render(
        "apply",
        context! {
            captcha_sitekey: hc.sitekey.as_str()
        },
    )
}

#[post("/apply", data = "<data>")]
async fn post_apply(data: Form<VmRequest>, hc: &State<HCaptchaParams>) -> Option<Redirect> {
    let captcha = reqwest::Client::new()
        .post("https://hcaptcha.com/siteverify")
        .query(&[
            ("secret", &hc.secret),
            ("response", &data.h_captcha_response),
        ])
        .send()
        .await
        .ok()?;

    if !captcha.status().is_success() {
        return None;
    }

    Some(Redirect::to("/success"))
}

#[get("/success")]
fn get_success() -> Template {
    Template::render("success", context! {})
}

#[launch]
fn rocket() -> _ {
    dotenv::dotenv().ok();
    env_logger::init();

    let captcha = HCaptchaParams::from_env();

    rocket::build()
        .attach(Template::fairing())
        .manage(captcha)
        .mount("/", routes![get_apply, post_apply, get_success])
}
