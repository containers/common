#!/usr/bin/env bash
# 
# rebuild_quay_ctr_images.sh PROJECT AUTHFILE
#
# This script expects Podman to be installed on the system and
# that the user is logged in with `podman login` to the target
# repository.
#
# This script expects one or two parameters, it will query for 
# the PROJECT if not provided.  It expects these parameters
# in order:
#     PROJECT  (buildah, podman or skopeo)
#     AUTHFILE (Location of the authfile created by podman login)
#
# It then builds the quay.io/container/${PROJECT}:latest, 
# quay.io/${PROJECT}/stable:latest, quay.io/${PROJECT}/upstream:latest,
# and the quay.io/${PROJECT}/testing:latest container images.
# It then pushes the images to those repos.
#
# This script can be run via cron to build and push the images
# on a regular basis.  Entries like this within "crontab -e" would 
# create each of the images for the specified project at 8:00,
# 9:00 and 10:00 a.m. successively.
# The entries assume an auth.json file has been created using the
# `podman login quay.io` command as root.  If run as a rootless
# user the '0' in the location would need to be changed to the uid
# of the user.  This also assumes a copy of this file in the
# /root/quay.io directory.
#
#    0 8 * * * /root/quay.io/rebuild_quay_ctr_images.sh buildah /run/user/0/containers/auth.json 
#    0 9 * * * /root/quay.io/rebuild_quay_ctr_images.sh skopeo /run/user/0/containers/auth.json
#    0 10 * * * /root/quay.io/rebuild_quay_ctr_images.sh podman /run/user/0/containers/auth.json
#

PROJECT=$1
AUTHFILE=$2

########
# Query for project and authfile 
# if necessary.
########
if [[ "$PROJECT" == "" ]]; then
  echo "Enter the containers project to create the images for:"
  read PROJECT
  echo ""
fi

if [[ "$AUTHFILE" == "" ]]; then
  echo "Enter the authfile to use:"
  read AUTHFILE
  echo ""
fi

GITHUBPROJECT=$PROJECT
# Tweak until podman lives under podman and not libpod in GitHub
if [[ "$PROJECT" == "podman" ]]; then
   GITHUBPROJECT="libpod"
fi  

########
# Build quay.io/containers/${PROJECT}:latest
########
podman build --no-cache -t quay.io/containers/${PROJECT}:latest -f https://raw.githubusercontent.com/containers/${GITHUBPROJECT}/master/contrib/${PROJECT}image/stable/Dockerfile .

########
# Push quay.io/containers/${PROJECT}:latest
########
podman push --authfile ${AUTHFILE} quay.io/containers/${PROJECT}:latest

########
# Remove quay.io/containers/${PROJECT}:latest
########
podman rmi -f quay.io/containers/${PROJECT}:latest

for REPO in stable upstream testing
do
########
# Build quay.io/${PROJECT}/${REPO}:latest
########
podman build --no-cache -t quay.io/${PROJECT}/${REPO}:latest -f https://raw.githubusercontent.com/containers/${GITHUBPROJECT}/master/contrib/${PROJECT}image/${REPO}/Dockerfile .

########
# Push quay.io/${PROJECT}/${REPO}:latest
########
podman push --authfile ${AUTHFILE} quay.io/${PROJECT}/${REPO}:latest

########
# Remove quay.io/${PROJECT}/${REPO}:latest
########
podman rmi -f quay.io/${PROJECT}/${REPO}:latest

done

########
# That's All Folks!!!
########
