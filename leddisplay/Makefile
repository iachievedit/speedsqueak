APP=leddisplay

LIBS=-lwiringPi -lzmq -ljansson -lm

$(APP):	$(APP).c
	gcc -o $(APP) $< $(LIBS)

clean:
	rm -rf $(APP)