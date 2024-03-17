use rocket::{
    http::Status,
    response::{self, status, Responder},
    Request,
};
use rocket_dyn_templates::{context, Template};

pub type Res<T> = Result<T, Error>;

#[derive(Debug)]
pub struct Error {
    reference: uuid::Uuid,
    #[allow(unused)]
    error: anyhow::Error,
}

impl<E> From<E> for Error
where
    E: Into<anyhow::Error>,
{
    fn from(error: E) -> Self {
        Self {
            reference: uuid::Uuid::new_v4(),
            error: error.into(),
        }
    }
}

impl<'r> Responder<'r, 'static> for Error {
    fn respond_to(self, request: &Request<'_>) -> response::Result<'static> {
        // log a UUID and display the UUID in the 500 handler
        log::warn!("{self:#?}");
        let templ = Template::render("500", context! {reference: self.reference.to_string()});
        status::Custom(Status::InternalServerError, templ).respond_to(request)
    }
}
