#!/bin/bash

go get ./wikiracer
go install ./wikiracer

bold=$(tput bold)
red=$(tput setaf 1)
reset=$(tput sgr0)

assert_output_equals() {
    if [[ "$1" == "$2" ]]; then
        echo "${bold}Assertion passed: output did equal: ${2}${reset}"
    else
        echo "${bold}${red}Assertion failed: output did not equal: ${2}${reset}"
        exit 1
    fi
}

# Test output when one of the articles is a dead-end
assert_output_equals \
    "$(wikiracer find "Mike Tyson" "User:Ucarion" 2>/dev/null)" \
    "No path found."

assert_output_equals \
    "$(wikiracer find "Mike Tyson" "User:Ucarion" --format=json 2>/dev/null)" \
    "null"

# Test trivial zero-step path
assert_output_equals \
    "$(wikiracer find "Mike Tyson" "Mike Tyson" 2>/dev/null)" \
    "Mike Tyson"

assert_output_equals \
    "$(wikiracer find "Mike Tyson" "Mike Tyson" --format=json 2>/dev/null)" \
    "[\"Mike Tyson\"]"

# TODO Test one-step paths? I haven't found a reliable real-world way to produce
# them.

# Test two-step path
assert_output_equals \
    "$(wikiracer find "Kevin Bacon" "Paul Erdős" 2>/dev/null)" \
    "Kevin Bacon -> Erdős number -> Paul Erdős"

assert_output_equals \
    "$(wikiracer find "Kevin Bacon" "Paul Erdős" --format=json 2>/dev/null)" \
    "[\"Kevin Bacon\",\"Erdős number\",\"Paul Erdős\"]"

# Test title normalization and URLs
assert_output_equals \
    "$(wikiracer find "en.wikipedia.org/wiki/Albert_Einstein" "albert Einstein" 2>/dev/null)" \
    "Albert Einstein"
