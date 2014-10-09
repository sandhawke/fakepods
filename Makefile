
default:
	@echo you have to say what to make

snapshot:
	curl -o dump.json 'http://nonesuch.fakepods.com/_nearby'
	mkdir -p snapshots
	cp dump.json snapshots/`date +%Y%m%dT%H:%M:%S`.json 

# maybe one should just use two server, so it can be port 80 on both?

# be sure to install and run rcconf to actually set the runlevels
# for boot time

dev-deploy:
	scp debian-init-script-dev root@fakepods.com:/etc/init.d/devfakepods
	scp *.go root@fakepods.com:/root/go/src/github.com/sandhawke/devfakepods
	ssh root@fakepods.com "cd go/src/github.com/sandhawke/devfakepods && go build && go test && go install && /etc/init.d/devfakepods restart"

production-deploy:
	scp debian-init-script root@fakepods.com:/etc/init.d/fakepods
	scp *.go root@fakepods.com:/root/go/src/github.com/sandhawke/fakepods
	ssh root@fakepods.com "cd go/src/github.com/sandhawke/fakepods && go build && go test && go install && /etc/init.d/fakepods restart"

