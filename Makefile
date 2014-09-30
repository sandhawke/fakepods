
default:
	@echo you have to say what to make

# not sure why debian's daemon scripts don't seem to be able to start this
# so... here's a hack for now.
deploy:
	scp fakepods.go root@fakepods.com:
	ssh root@fakepods.com "go build fakepods.go & mv fakepods /usr/local/sbin"
	curl -o dump.json 'http://nonesuch.fakepods.com/**'
	mkdir -p snapshots
	cp dump.json snapshots/`date +%Y%m%dT%H:%M:%S`.json 
	ssh -f root@fakepods.com "killall fakepods; nohup /usr/local/sbin/fakepods --root &"
	sleep 1
	curl -X PUT -d @dump.json 'http://fakepods.com/**'

