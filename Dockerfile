FROM rust:latest
RUN mkdir /app
WORKDIR /app
ENV ROCKET_PORT=8000
EXPOSE $ROCKET_PORT

COPY ./ /app
RUN cargo build --release
CMD ["./target/release/vsos-app"]
