APP=leddisplay

LIBS=-lwiringPi -lzmq -ljansson -lm

$(APP):	$(APP).c
	gcc -o $(APP) $< $(LIBS)

install:
	sudo install $(APP) /usr/local/bin

clean:
	rm -rf $(APP)
