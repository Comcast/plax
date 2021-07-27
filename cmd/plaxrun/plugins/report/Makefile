
DIRS := $(sort $(dir $(wildcard */)))

all: $(DIRS)
	@echo DIRS=${DIRS}

.PHONY: $(DIRS)

$(DIRS):
	echo dir=$@
	cd $@; make
