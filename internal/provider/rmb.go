package provider

/* TODO: unused
func startRmbIfNeeded(ctx context.Context, api *apiClient) {
	const RMBWorkers = 10

	if api.use_rmb_proxy {
		return
	}
	rmbClient, err := gormb.NewServer(api.manager, "127.0.0.1:6379", RMBWorkers, api.identity)
	if err != nil {
		log.Fatalf("couldn't start server %s\n", err)
	}
	if err := rmbClient.Serve(ctx, api.manager); err != nil {
		log.Printf("error serving rmb %s\n", err)
	}
}
*/
