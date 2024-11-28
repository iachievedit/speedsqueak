install:
	sudo cp speedsqueak-camera.sh /usr/local/bin
	sudo cp speedsqueak-uploader.sh /usr/local/bin
	sudo cp speedsqueak-heartbeat.sh /usr/local/bin
	sudo cp speedsqueak-ops243.sh /usr/local/bin
	sudo cp speedsqueak-camera.service /etc/systemd/system/
	sudo cp speedsqueak-uploader.service /etc/systemd/system/
	sudo cp speedsqueak-heartbeat.service /etc/systemd/system/
	sudo cp speedsqueak-radar.service /etc/systemd/system/
	sudo cp speedsqueak-leddisplay.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable speedsqueak-camera
	sudo systemctl enable speedsqueak-radar
	sudo systemctl enable speedsqueak-uploader
	sudo systemctl enable speedsqueak-heartbeat
	sudo systemctl enable speedsqueak-leddisplay
	


