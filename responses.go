package main

const (
	code150        = "150 File status okay; about to open data connection."
	code200        = "200 Command okay."
	code202        = "202 Command not implemented, superfluous at this site."
	code220        = "220 Service ready for new user."
	code221        = "221 Service closing control connection."
	code226        = "226 Closing data connection."
	code227        = "227 Entering Passive Mode (%v,%v,%v,%v,%v,%v)."
	code230        = "230 User logged in, proceed."
	code234        = "234 Starting TLS negociation."
	code250        = "250 Requested file action okay, completed."
	code257created = "257 \"%v\" created."
	code257        = "257 \"%v\""
	code331        = "331 User name okay, need password."
	code350        = "350 Requested file action pending further information."
	code421        = "421 Service not available, closing control connection."
	code425        = "425 Can't open data connection."
	code426        = "426 Connection closed; transfer aborted."
	code450        = "450 Requested file action not taken."
	code500        = "500 Syntax error, command unrecognized."
	code502        = "502 Command not implemented."
	code501        = "501 Syntax error in parameters or arguments."
	code504        = "504 Command not implemented for that parameter."
	code530        = "530 Not logged in."
	code550        = "550 Requested action not taken."
	code553        = "553 Requested action not taken."
)

const (
	ascii    = "A"
	image    = "I"
	nonPrint = "N"
)

const (
	stream = "S"
)

const (
	file = "F"
)
