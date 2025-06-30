fn main() {
    tonic_build::configure()
        .build_server(false)
        .compile(&["../common/pb/repo.proto"], &["../common/pb"])
        .unwrap();
}

