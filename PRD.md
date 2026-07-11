# Product Requirements Document

## Executive Summary

A simple, shared web application for open dice rolling, to be used by table-top role players.

## Problem Statement

When playing in person, players can rol physical dice on the table in front of everyone. When playing over Zoom, Meet, or Discord, that open dice rolling is no longer easily possible. There are services that allow some limited set of dice rolling options, including Google Chat (i.e. '/roll 2d20+2'), but it lacks a lot of features.

## User Stories & Requirements

- All players, including the game master:
    - Login and get the option join an existing room or create a new one
    - If they choose to create a new room, a unique room identifier string is generated and this is used to construct a URL which can then be shared with all other players
    - If they choose to enter an existing room, they can either use the generated URL that was shared with them, or they can enter the unique identifier into the text box and go
    - Once in the room, they see the whole chat history since the creation of the room
    - They have the ability to chat plain text and have it appear in the history and shared with all other players
    - They have the ability to roll dice, with the results going to the chat history and shared with all other players
    - should be able to roll any combination of polyhedral dice and everyone involved should be able to confirm the results
    - Dice rolling as a function with parameters, that allow complex arrangement's 
    - Set of buttons for common dice combinations:
        - 1d20
        - max(2d20) "d20 with advantage"
        - min(2d20) "d20 with disadvantage"
        - d100
    - Ability to create and test new combinations and then save that to a new button, as a macro
        - Custom buttons are per-user-per-room; everytime that user accesses that room they see the buttons they defined there
- Administrator
    - Has ability to see and enter any room
    - Has the ability to prune old chat history, as needed
    - Has the ability to delete unused rooms
    - Can view database table of all registered users, and easily see who is online

## Technical Considerations

- Very low expected usage
    - Less than 50 users
    - Either once a week or once every two weeks
- Application written in Go with SQLite for storage and persistance
- HTML/CSS/HTMX for the web interface
- Users should be able to register using social login
    - Starting with Google

## Dice Rolling Domain Specific Language

- `NdS` where `N` is an integer denoting the number of dice to roll and `S` is the number of sides of each dice
    - When not specified `N` is equal to 1
    - When not specified `S` is equal to 6
    - Returns an array of random numbers from 1 to `S`
    - `dF` is a special case, "Fudge Dice," and represents a six sided die where the faces are [+1, +1, 0, 0, -1, -1]
    - Examples:
        - `1d` := 1 six sided die
        - `d` := 1 six sided die
        - `d20` := 1 twenty sided die
        - `3d6` := 3 six sided dice
        - `2dF` := 2 "Fudge" dice as described above
- `,` allows chaining dice combinations or functions together to produce an array
    - Examples:
        - `1,2,3` := the array [1, 2, 3]
        - `2d,d4` := the array made up of the results of rolling 2 six sided dice and 1 four sided die, the results will contain 3 random numbers
        - `3d6,4` := the array made up of the results of rolling 3 six sided dice and the integer 4; the results will contain 3 random integers and the fixed integer `4`
- `sum()` "add all array elements"
```
define sum(param as array) {
    var ACC = 0
    foreach var p in param:
        ACC += p
    return ACC
}
```
- `max()` "return largest array element"
```
define max(param as array) {
    var ACC = 0
    foreach var p in param:
        if( p > ACC ){ ACC = p }
    return ACC
}
```
- `min()` "return smallest array element"
```
define min(param as array) {
    var ACC = 99999
    foreach var p in param:
        if( p < ACC ){ ACC = p }
    return ACC
}
```
- `maxk()` "return largest k elements from array"
```
define maxk(param as array) {
    var K = shift param
    if( K > length(param) ){ K = length(param) }
    var list as array
    list = reverse(sort(param))
    while( length(list) > K ){
        pop(list)
    }
    return list
}
```
- `mink()` "return smallest k elements from array"
```
define mink(param as array) {
    var K = shift param
    if( K > length(param) ){ K = length(param) }
    var list as array
    list = sort(param)
    while( length(list) > K ){
        pop(list)
    }
    return list
}
```
- `count()` "return count of elements in array"
```
define count(param as array){
    var ACC = 0
    foreach var p in param:
        ACC++
    return ACC
}
```
- `equals()` "return elements equal to a value"
```
define equals(param as array){
    var K = shift param
    var list as array
    foreach var p in param:
        if( p == l ){ push(list, p) }
    return list
}
```
- `above()` "return elements greater than a value"
```
define above(param as array){
    var K = shift param
    var list as array
    foreach var p in param:
        if( p > k ){ push(list, p) }
    return list
}
```
- `below()` "return elements less than a value"
```
define below(param as array){
    var K = shift param
    var list as array
    foreach var p in param:
        if( p < k ){ push(list, p) }
    return list
}
```
- `+, -, *, /` "normal math operators, as they are traditionally used"

## Success Criteria/Metrics

- Starting the application brings up a custom web server on a non-standard port (which can be proxied on the hosting server)
- New users can register with Google logins
- Users, once registered, can create or join rooms
- Once users have joined a room, they can chat (text) and roll dice
- The text chats and the dice rolls are persisted in the chat history for that room, and shared among all other users accessing the room

## Scope

- Single compiled Go executable
- SQLite, saved to the servers drive storage
- Maintains it's own list of registered users, but auth is handled by Google login
