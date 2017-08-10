#!/bin/sh

run_app() {
  consul-snapshot "${@}"
}

case "${1}" in
  'backup')
    run_app "${1}"
    ;;

  'restore')
    run_app "${@}"
    ;;

  *)
    exec "${@}"
    ;;

esac
