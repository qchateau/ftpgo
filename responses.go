package main

const (
	okOpenDt                = "150 File status okay; about to open data connection."
	commandOk               = "200 Command okay."
	commandNotImplemented   = "202 Command not implemented, superfluous at this site."
	readyForNewUser         = "220 Service ready for new user."
	loggedOut               = "221 Service closing control connection."
	okCloseDt               = "226 Closing data connection."
	loggedIn                = "230 User logged in, proceed."
	actionOk                = "250 Requested file action okay, completed."
	created                 = "257 \"%v\" created."
	currentDir              = "257 \"%v\""
	closing                 = "421 Service not available, closing control connection."
	actionNotTaken          = "450 Requested file action not taken."
	syntaxError             = "500 Syntax error, command unrecognized."
	parameterNotImplemented = "504 Command not implemented for that parameter."
	notLoggedIn             = "530 Not logged in."
	passiveMode             = "227 Entering Passive Mode (%v,%v,%v,%v,%v,%v)."
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
