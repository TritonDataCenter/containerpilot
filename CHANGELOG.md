## 3.4.2 (Unreleased)

## 3.4.1 (August 21st, 2017)

BUG FIXES:

- move defer out of loop so as not to leak a closure (#488)

SHA1 4d13cfb345de86135ab2271b77516c6b6a7bed3a

## 3.4.0 (August 18, 2017)

FEATURES:

- config: Introduce logging to file w/ log file re-open on SIGUSR1 (#477)
- add raw logging field to bypass logger for exec (#462)

BUG FIXES:

- control: HTTPServer should handle existing control socket files (#480)
- docs: Better language around `stopTimeout` (#479)
- fix GOOS setting in makefile (#483)
- discovery: fix tls config for Consul (#481)
- default restarts to "unlimited" when when->interval is set (#473)
- docs: add pointers to godoc (#475)

SHA1 ff14bfc9f6b7a10654b0c8777175c2b0436575aa

## 3.3.4 (August 9, 2017)

BUG FIXES:

* fix race that can sometimes cause deadlock during reload/shutdown with larger numbers of jobs (#468) (#469)

SHA1 806f28a25a06acdbcfa8940c8968d5f8e20a2c4f

## 3.3.3 (August 8, 2017)

BUG FIXES:

- make sure jobs configured for stopping/stopped exit on shutdown/reload (#465) (#466)

SHA1 8d680939a8a5c8b27e764d55a78f5e3ae7b42ef4

## 3.3.2 (August 2, 2017)

BUG FIXES:

- Fix when->timeout canceling running jobs (#456) (#458)

SHA1 056d45f728e9b9c61793d6f994da291d5eebeabd

## 3.3.1 (July 31, 2017)

BUG FIXES:

- fixed bug where `/status` always reported job as "unknown" status (#445) (#450)
- fixed bug where job exec was getting `SIGKILL` instead of `SIGTERM` on ContainerPilot stop (#448) (#449)
- fixed bug where supervisor's `SIGCHLD` handler could block `SIGTERM`/`SIGINT` handlers

SHA1 e27c1b9cd1023e622f77bb19914606dee3c9b22c

## 2.7.7 (July 31, 2017)

BUG FIXES:

- Backport supervisor process to v2 to avoid race in zombie cleanup after timeout (#447) (#452)

SHA1 030f1e54a43a842d38b30373f8847132a9771829

## 3.3.0 (July 19, 2017)

BUG FIXES:

- move child reaping into supervisor process (#439) (#440)
- bugfix for catching another when event stopping the running job (#417) (#438)
