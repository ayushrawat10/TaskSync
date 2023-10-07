package main

import (
    "fmt"
    "context"
    "errors"
    "net/http"
    "time"
    "os"
    "os/signal"
    "syscall"
)

func (app *application) serve() error {
    srv := &http.Server{
        Addr: fmt.Sprintf(":%d", app.config.port),
        Handler: app.routes(),
        IdleTimeout: time.Minute,
        ReadTimeout: 10*time.Second,
        WriteTimeout: 30*time.Second,
    }

    shutdownError := make(chan error)

    go func() {
        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        s := <-quit
        app.logger.PrintInfo("Shutting down server", map[string]string{
            "signal": s.String(),
        })

        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        err := srv.Shutdown(ctx)
        if err != nil {
            shutdownError <- err
        }

        app.logger.PrintInfo("completing background tasks", map[string]string{
            "addr": srv.Addr,
        })
        app.wg.Wait()
        shutdownError <- nil
    }()

    app.logger.PrintInfo("Starting server", map[string]string{
        "addr": srv.Addr,
        "env": app.config.env,
    })

    err := srv.ListenAndServe()
    if !errors.Is(err, http.ErrServerClosed) {
        return err
    }

    err = <-shutdownError
    if err != nil {
        return err
    }

    app.logger.PrintInfo("Stopped server", map[string]string{
        "addr": srv.Addr,
    })

    return nil
}
