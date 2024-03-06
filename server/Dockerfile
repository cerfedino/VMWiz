FROM rust:latest
RUN mkdir /app
WORKDIR /app
ENV ROCKET_PORT=8000
EXPOSE 8000

COPY ./ /app
RUN cargo build --release
CMD ["./target/release/vsos-app"]
