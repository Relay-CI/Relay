package main

import (
	"net/http"
	nhpprof "net/http/pprof"
)

// registerControlRoutes owns the routes shared by the network and local-socket
// transports, including the authorization policy attached to each handler.
func (s *Server) registerControlRoutes(mux *http.ServeMux) {
	authAny := s.auth
	authOwner := s.authWithRoles("owner")
	authDeployer := s.authWithRoles("owner", "admin", "deployer")
	authReadOnly := s.authByMethod(nil, nil)
	authReadDeployerWrite := s.authByMethod(nil, []string{"owner", "admin", "deployer"})

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.HandleFunc("/api/public/theme", s.handlePublicTheme)
	mux.HandleFunc("/api/auth/session", s.handleDashboardSession)
	mux.HandleFunc("/api/auth/setup", s.handleAuthSetup)
	mux.HandleFunc("/api/auth/login", s.handleAuthLogin)
	mux.HandleFunc("/api/auth/me", s.handleAuthMe)
	mux.HandleFunc("/api/auth/cli/start", s.handleAuthCLIStart)
	mux.HandleFunc("/api/auth/cli/exchange", s.handleAuthCLIExchange)
	mux.HandleFunc("/api/deploys", authReadDeployerWrite(s.handleDeploys))
	mux.HandleFunc("/api/deploys/cancel/", authDeployer(s.handleDeployCancel))
	mux.HandleFunc("/api/deploys/", authAny(s.handleDeployByID))
	mux.HandleFunc("/api/deploys/rollback", authDeployer(s.handleRollback))
	mux.HandleFunc("/api/apps/start", authDeployer(s.handleAppStart))
	mux.HandleFunc("/api/apps/stop", authDeployer(s.handleAppStop))
	mux.HandleFunc("/api/apps/delete-lane", authDeployer(s.handleLaneDelete))
	mux.HandleFunc("/api/apps/restart", authDeployer(s.handleAppRestart))
	mux.HandleFunc("/api/apps/config", authReadDeployerWrite(s.handleAppConfig))
	mux.HandleFunc("/api/apps/signed-link", authDeployer(s.handleSignedLink))
	mux.HandleFunc("/api/server/config", authOwner(s.handleServerConfig))
	mux.HandleFunc("/api/apps/companions", authReadDeployerWrite(s.handleAppCompanions))
	mux.HandleFunc("/api/apps/companions/restart", authDeployer(s.handleCompanionRestart))
	mux.HandleFunc("/api/apps/secrets", authDeployer(s.handleAppSecrets))
	mux.HandleFunc("/api/plugins/buildpacks", authOwner(s.handleBuildpackPlugins))
	mux.HandleFunc("/api/plugins/buildpacks/", authOwner(s.handleBuildpackPluginByName))
	mux.HandleFunc("/api/plugins/buildpacks/install-url", authOwner(s.handleBuildpackPluginInstallURL))
	mux.HandleFunc("/api/plugins/catalog", authOwner(s.handleBuildpackPluginCatalog))
	mux.HandleFunc("/api/admin/ops", authOwner(s.handleAdminOps))
	mux.HandleFunc("/api/logs/", authAny(s.handleLogsByID))
	mux.HandleFunc("/api/logs/stream/", authAny(s.handleLogsStream))
	mux.HandleFunc("/api/runtime/logs/targets", authAny(s.handleRuntimeLogTargets))
	mux.HandleFunc("/api/runtime/logs/stream", authAny(s.handleRuntimeLogStream))
	mux.HandleFunc("/api/events", authAny(s.handleEvents))
	mux.HandleFunc("/api/sync/start", authDeployer(s.handleSyncStart))
	mux.HandleFunc("/api/sync/plan/", authDeployer(s.handleSyncPlan))
	mux.HandleFunc("/api/sync/upload/", authDeployer(s.handleSyncUpload))
	mux.HandleFunc("/api/sync/bundle/", authDeployer(s.handleSyncBundle))
	mux.HandleFunc("/api/sync/delete/", authDeployer(s.handleSyncDelete))
	mux.HandleFunc("/api/sync/finish/", authDeployer(s.handleSyncFinish))
	mux.HandleFunc("/api/sync/pull/", authDeployer(s.handleSyncPull))
	mux.HandleFunc("/api/projects", authReadOnly(s.handleProjects))
	mux.HandleFunc("/api/projects/delete", authOwner(s.handleProjectDelete))
	mux.HandleFunc("/api/promotions", authReadDeployerWrite(s.handlePromotions))
	mux.HandleFunc("/api/promotions/approve", authOwner(s.handlePromotionApprove))
	mux.HandleFunc("/api/webhooks/github", s.handleGithubWebhook)
	mux.HandleFunc("/api/edge/authz", s.handleEdgeAuthz)
	mux.HandleFunc("/api/doctor", authAny(s.handleDoctor))
	mux.HandleFunc("/api/users", authOwner(s.handleUsers))
	mux.HandleFunc("/api/users/", authOwner(s.handleUserByID))
	mux.HandleFunc("/api/audit", authOwner(s.handleAuditLog))
}

// registerNetworkControlRoutes adds routes intentionally available only on the
// network transport, preserving the existing transport interface.
func (s *Server) registerNetworkControlRoutes(mux *http.ServeMux) {
	s.registerControlRoutes(mux)
	authAny := s.auth
	authOwner := s.authWithRoles("owner")
	authDeployer := s.authWithRoles("owner", "admin", "deployer")
	mux.HandleFunc("/api/deploys/rollout-abort", authDeployer(s.handleRolloutAbort))
	mux.HandleFunc("/api/analytics", authAny(s.handleAnalytics))
	if s.cloud != nil && s.cloud.hubEnabled {
		s.registerCloudHubRoutes(mux)
	}
	if getenvBool("RELAY_PPROF", false) {
		mux.HandleFunc("/debug/pprof/", authOwner(nhpprof.Index))
		mux.HandleFunc("/debug/pprof/cmdline", authOwner(nhpprof.Cmdline))
		mux.HandleFunc("/debug/pprof/profile", authOwner(nhpprof.Profile))
		mux.HandleFunc("/debug/pprof/symbol", authOwner(nhpprof.Symbol))
		mux.HandleFunc("/debug/pprof/trace", authOwner(nhpprof.Trace))
	}
}

func (s *Server) registerLocalControlRoutes(mux *http.ServeMux) {
	s.registerControlRoutes(mux)
}
