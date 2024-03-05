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
    Template::render(
        "apply",
        context! {
            captcha_sitekey: env.hcaptcha.sitekey.as_str()
        },
    )
}

#[post("/apply", data = "<data>")]
async fn post_apply(data: Form<VmRequest>, env: &State<Environment>) -> Option<Redirect> {
    let captcha = reqwest::Client::new()
        .post("https://hcaptcha.com/siteverify")
        .query(&[
            ("secret", &env.hcaptcha.secret),
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

    let env: Environment = serde_env::from_env().expect("failed to deserialize environment");

    rocket::build()
        .attach(Template::fairing())
        .manage(env)
        .mount("/", routes![get_apply, post_apply, get_success])
}
