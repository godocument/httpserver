# httpserver

a simple http server with signal handling using errgroup 

two goroutines are created, one for http server and the other for signal handling. if a ^C signal is receive, the 
signal handling goroutine will exit before notifying http server shutdown, thereafter the http server have a chance to 
exit. otherwise, if the http server runs into error, it will notify signal handling goroutine to quit through channel, 
before it returns.
