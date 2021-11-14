# httpserver

a simple http server with signal handling using errgroup 

serveral goroutines are created using errgroup Go method, one for signal handling, and others for http servers.

if a ^C signal is received, the signal handling goroutine will shutdown all http servers and then quit.

if any http server goroutine runs into error, it will shutdown all other http servers and notify signal handling 
goroutine to quit.
