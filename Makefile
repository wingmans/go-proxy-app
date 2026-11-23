.DEFAULT_GOAL := help

.PHONY: help

help:
	@printf '%s\n' \
		'Choose a proxy and run its Makefile:' \
		'' \
		'  make -C throttling help' \
		'  make -C authorization help'

%:
	@printf '%s\n' \
		'No proxy commands run from the repository root.' \
		'' \
		'Use one of:' \
		'  make -C throttling $@' \
		'  make -C authorization $@'
	@exit 2
