// This file is part of www.nand2tetris.org
// and the book "The Elements of Computing Systems"
// by Nisan and Schocken, MIT Press.
// File name: projects/04/Fill.asm

// Runs an infinite loop that listens to the keyboard input.
// When a key is pressed (any key), the program blackens the screen,
// i.e. writes "black" in every pixel;
// the screen should remain fully black as long as the key is pressed. 
// When no key is pressed, the program clears the screen, i.e. writes
// "white" in every pixel;
// the screen should remain fully clear as long as no key is pressed.

// Put your code here.
    @8192  // 32 (16ビットのワード32個で1行) x 512 (縦のピクセル数)
    D=A
    @MAXCOUNT  
    M=D
    @LOOP
    0;JMP

(SETSCREEN)
    @i
    D=M // D = i

    @MAXCOUNT
    D=D-M
    @LOOP
    D;JGE // i - MAXCOUNT >= 0 であれば､ LOOPへ移動

    @color
    D=M
    @pixelAddress
    A=M  // ピクセルのアドレスをセット
    M=D  // ピクセルの色をcolorの値にする
    @pixelAddress
    M=M+1

    @i
    M=M+1

    @SETSCREEN
    0;JMP

(WHITE)
    @color
    M=0

    @SETSCREEN
    0;JMP


(BLACK)
    @color
    M=-1

    @SETSCREEN
    0;JMP

(LOOP)
    @SCREEN
    D=A
    @pixelAddress // SCREENのアドレスを0としたときのピクセルのアドレス
    M=D

    @i
    M=0

    @KBD
    D=M

    @BLACK
    D;JNE // KBD != 0 であれば､ BLACKへ移動

    @WHITE
    0;JMP

