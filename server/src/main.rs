use ::serde::{Deserialize, Serialize};
use rocket::{
    form::Form, get, http::Status, launch, post, response::Redirect, routes, FromForm, State,
};
use rocket_dyn_templates::Template;
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
struct Mail {
    smtp_server: String,
    human_responder: String,
    sender: String,
    user: String,
    pass: String,
}

#[derive(Deserialize, Serialize, Debug, PartialEq, Eq, Clone)]
enum Deployment {
    Test,
    Prod,
}

impl Deployment {
    fn is_test(&self) -> bool {
        *self == Self::Test
    }
    fn is_prod(&self) -> bool {
        *self == Self::Prod
    }
}

#[derive(Deserialize, Serialize, Debug)]
struct Environment {
    hcaptcha: HCaptcha,
    mail: Mail,
    deployment: Deployment,
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

    match sendmail(data.into_inner(), &**env).await {
        Ok(..) => Ok(Redirect::to("/success")),
        Err(e) => {
            log::error!("failed to send e-mail ({e:?})");
            Err(Status::InternalServerError)
        }
    }
}

async fn sendmail(
    vm_request: VmRequest,
    env: &Environment,
) -> Result<(), Box<dyn 'static + std::error::Error>> {
    use lettre::message::header::ContentType;
    use lettre::message::Mailbox;
    use lettre::transport::smtp::authentication::Credentials;
    use lettre::*;

    let applicant = message::Mailbox {
        name: None,
        email: vm_request.ethz_email.parse()?,
    };

    let vsos_apply: Mailbox = env.mail.human_responder.parse().unwrap();

    let noreply = Mailbox {
        name: Some("vsos noreply".into()),
        email: env.mail.sender.parse().unwrap(),
    };

    let email = Message::builder()
        .from(noreply)
        .reply_to(vsos_apply.clone())
        .reply_to(applicant.clone())
        .to(vsos_apply)
        .cc(applicant)
        .subject(if env.deployment.is_test() {
            "[Test-Please-Ignore] VM Request"
        } else {
            "VM Request"
        })
        .header(ContentType::TEXT_PLAIN)
        .body(format_mail(&vm_request))?;

    let creds = Credentials::new(env.mail.user.clone(), env.mail.pass.clone());

    let mailer = SmtpTransport::starttls_relay(&env.mail.smtp_server)?
        .credentials(creds)
        .build();

    mailer.send(&email)?;

    Ok(())
}

fn format_mail(req: &VmRequest) -> String {
    let mut buf = String::new();

    buf += "This is a generated E-Mail to confirm your VM Request.\n";
    buf += "If you have not requested this or there are missing or incomplete information, please respond to this e-mail.\n";
    buf += "\n";
    buf += "\n";
    buf += &format!("OS: {}\n", req.os);
    buf += &format!("Hostname: {}\n", req.hostname);
    buf += &format!("RAM: {} MB\n", req.ram * 1024);
    buf += &format!("Disk: {}G\n", req.disk);
    buf += &format!("SSH keys:\n{}\n", req.ssh_keys);
    buf += &format!("University E-Mail: {}\n", req.ethz_email);
    buf += &format!("External E-Mail: {}\n", req.external_email);
    buf += &format!("Cores: {}\n", req.cores);

    if let Some(w) = &req.wishes {
        buf += &format!("requests: {w}\n");
    }

    buf
}

#[get("/success")]
fn get_success() -> Template {
    Template::render("success", ())
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
