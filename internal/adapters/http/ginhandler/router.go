package ginhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yoophi/codepush-server-golang/internal/adapters/http/httperrors"
	"github.com/yoophi/codepush-server-golang/internal/adapters/http/middleware"
	"github.com/yoophi/codepush-server-golang/internal/core/domain"
	"github.com/yoophi/codepush-server-golang/internal/core/ports"
)

func NewRouter(service ports.HTTPAPI) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to the CodePush REST API!")
	})
	router.GET("/health", func(c *gin.Context) {
		if err := service.Health(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "ERROR", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	router.GET("/auth/login", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html><body><h1>Login</h1></body></html>"))
	})
	router.GET("/auth/register", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html><body><h1>Register</h1></body></html>"))
	})
	for _, path := range []string{"/auth/login/github", "/auth/login/microsoft", "/auth/login/azure-ad", "/auth/callback/github"} {
		router.GET(path, func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "oauth flow not implemented"})
		})
	}

	router.GET("/updateCheck", func(c *gin.Context) {
		req := domain.UpdateCheckRequest{
			DeploymentKey:  c.Query("deploymentKey"),
			AppVersion:     firstNonEmpty(c.Query("appVersion"), c.Query("app_version")),
			PackageHash:    firstNonEmpty(c.Query("packageHash"), c.Query("package_hash")),
			Label:          c.Query("label"),
			ClientUniqueID: firstNonEmpty(c.Query("clientUniqueId"), c.Query("client_unique_id")),
		}
		resp, err := service.UpdateCheck(c.Request.Context(), req)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"updateInfo": resp})
	})
	router.GET("/v0.1/public/codepush/update_check", func(c *gin.Context) {
		req := domain.UpdateCheckRequest{
			DeploymentKey:  c.Query("deploymentKey"),
			AppVersion:     firstNonEmpty(c.Query("appVersion"), c.Query("app_version")),
			PackageHash:    firstNonEmpty(c.Query("packageHash"), c.Query("package_hash")),
			Label:          c.Query("label"),
			ClientUniqueID: firstNonEmpty(c.Query("clientUniqueId"), c.Query("client_unique_id")),
		}
		resp, err := service.UpdateCheck(c.Request.Context(), req)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"update_info": resp})
	})
	router.POST("/reportStatus/deploy", func(c *gin.Context) {
		var req domain.DeploymentStatusReport
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, domain.ErrMalformedRequest)
			return
		}
		if err := service.ReportDeploy(c.Request.Context(), req); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusOK)
	})
	router.POST("/v0.1/public/codepush/report_status/deploy", func(c *gin.Context) {
		var req domain.DeploymentStatusReport
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, domain.ErrMalformedRequest)
			return
		}
		if err := service.ReportDeploy(c.Request.Context(), req); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusOK)
	})
	router.POST("/reportStatus/download", func(c *gin.Context) {
		var req domain.DownloadReport
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, domain.ErrMalformedRequest)
			return
		}
		if err := service.ReportDownload(c.Request.Context(), req); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusOK)
	})
	router.POST("/v0.1/public/codepush/report_status/download", func(c *gin.Context) {
		var req domain.DownloadReport
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, domain.ErrMalformedRequest)
			return
		}
		if err := service.ReportDownload(c.Request.Context(), req); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusOK)
	})

	authed := router.Group("/")
	authed.Use(middleware.RequireAuth(service))

	authed.GET("/authenticated", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"authenticated": true})
	})
	authed.GET("/account", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.GetAccount(c.Request.Context(), account.ID)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"account": value})
	})
	authed.GET("/accessKeys", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.ListAccessKeys(c.Request.Context(), account.ID)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"accessKeys": value})
	})
	authed.POST("/accessKeys", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		var req domain.AccessKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, domain.ErrMalformedRequest)
			return
		}
		value, err := service.CreateAccessKey(c.Request.Context(), account.ID, c.ClientIP(), req)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"accessKey": value})
	})
	authed.GET("/accessKeys/:accessKeyName", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.GetAccessKey(c.Request.Context(), account.ID, c.Param("accessKeyName"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"accessKey": value})
	})
	authed.PATCH("/accessKeys/:accessKeyName", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		var req domain.AccessKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, domain.ErrMalformedRequest)
			return
		}
		value, err := service.UpdateAccessKey(c.Request.Context(), account.ID, c.Param("accessKeyName"), req)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"accessKey": value})
	})
	authed.DELETE("/accessKeys/:accessKeyName", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		if err := service.DeleteAccessKey(c.Request.Context(), account.ID, c.Param("accessKeyName")); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	authed.DELETE("/sessions/:createdBy", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		if err := service.DeleteSessionsByCreator(c.Request.Context(), account.ID, c.Param("createdBy")); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	authed.GET("/apps", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.ListApps(c.Request.Context(), account.ID)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"apps": value})
	})
	authed.POST("/apps", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		var req domain.AppCreationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, domain.ErrMalformedRequest)
			return
		}
		value, err := service.CreateApp(c.Request.Context(), account.ID, req)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"app": value})
	})
	authed.GET("/apps/:appName", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.GetApp(c.Request.Context(), account.ID, c.Param("appName"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"app": value})
	})
	authed.PATCH("/apps/:appName", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		var req domain.AppPatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, domain.ErrMalformedRequest)
			return
		}
		value, err := service.UpdateApp(c.Request.Context(), account.ID, c.Param("appName"), req)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"app": value})
	})
	authed.DELETE("/apps/:appName", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		if err := service.DeleteApp(c.Request.Context(), account.ID, c.Param("appName")); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	authed.POST("/apps/:appName/transfer/:email", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		if err := service.TransferApp(c.Request.Context(), account.ID, c.Param("appName"), c.Param("email")); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusCreated)
	})
	authed.POST("/apps/:appName/collaborators/:email", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		if err := service.AddCollaborator(c.Request.Context(), account.ID, c.Param("appName"), c.Param("email")); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusCreated)
	})
	authed.GET("/apps/:appName/collaborators", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.ListCollaborators(c.Request.Context(), account.ID, c.Param("appName"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"collaborators": value})
	})
	authed.DELETE("/apps/:appName/collaborators/:email", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		if err := service.RemoveCollaborator(c.Request.Context(), account.ID, c.Param("appName"), c.Param("email")); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	authed.GET("/apps/:appName/deployments", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.ListDeployments(c.Request.Context(), account.ID, c.Param("appName"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployments": value})
	})
	authed.POST("/apps/:appName/deployments", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		var req domain.DeploymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, domain.ErrMalformedRequest)
			return
		}
		value, err := service.CreateDeployment(c.Request.Context(), account.ID, c.Param("appName"), req)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"deployment": value})
	})
	authed.GET("/apps/:appName/deployments/:deploymentName", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.GetDeployment(c.Request.Context(), account.ID, c.Param("appName"), c.Param("deploymentName"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment": value})
	})
	authed.PATCH("/apps/:appName/deployments/:deploymentName", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		var req domain.DeploymentPatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, domain.ErrMalformedRequest)
			return
		}
		value, err := service.UpdateDeployment(c.Request.Context(), account.ID, c.Param("appName"), c.Param("deploymentName"), req)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deployment": value})
	})
	authed.DELETE("/apps/:appName/deployments/:deploymentName", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		if err := service.DeleteDeployment(c.Request.Context(), account.ID, c.Param("appName"), c.Param("deploymentName")); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	authed.GET("/apps/:appName/deployments/:deploymentName/history", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.GetHistory(c.Request.Context(), account.ID, c.Param("appName"), c.Param("deploymentName"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"history": value})
	})
	authed.DELETE("/apps/:appName/deployments/:deploymentName/history", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		if err := service.ClearHistory(c.Request.Context(), account.ID, c.Param("appName"), c.Param("deploymentName")); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	authed.GET("/apps/:appName/deployments/:deploymentName/metrics", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.GetMetrics(c.Request.Context(), account.ID, c.Param("appName"), c.Param("deploymentName"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"metrics": value})
	})
	authed.POST("/apps/:appName/deployments/:deploymentName/rollback", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.Rollback(c.Request.Context(), account.ID, c.Param("appName"), c.Param("deploymentName"), "")
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"package": value})
	})
	authed.POST("/apps/:appName/deployments/:deploymentName/rollback/:targetRelease", func(c *gin.Context) {
		account := middleware.CurrentAccount(c)
		value, err := service.Rollback(c.Request.Context(), account.ID, c.Param("appName"), c.Param("deploymentName"), c.Param("targetRelease"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"package": value})
	})

	return router
}

func writeError(c *gin.Context, err error) {
	c.JSON(httperrors.Status(err), gin.H{"error": err.Error()})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
