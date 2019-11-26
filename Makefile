###############################################
#
# Makefile
#
###############################################

http:
	go run main.go -proto http

https:
	go run main.go -key server.key -cert server.crt -proto https