# Jagdfragenpruefer fuer Bayern

## Base architecture

### Pre-process step
Golang based code opens all local pdfs based on their naming: fragenkatalog_2025_by.pdf. <basic name>_<year>_state.pdf.

The golang code reads the pdf file which contains over 1k questions and parses them out with the answer options and the right answers. The output is a json file.

### The Webapplication

The json file is the input for a single page application.

The single page application is based on simple javascript (worst case most popular js library for simplicty) on a single html website.

The goal of the page is to do basic space repetition on these questions and answers. With the answer prsented to you being in a random order every time you see/learn the question/answer.

The application can be used as a PWA and uses local storage only.

the state can be imported and exported through a basic button wereever the butotn makes sens.

The flow is very simple:
- You open the app on a browser or mobile phone
- You see a basic statistic about number of questions, number of questions learned fully and number of questions in progress of learning
- You press a button continue,  5, 10 or 15 (the button can be configured simply) and the number only mean the block of cards you want to learn right now.
  - The continue button only teaches you the cards which are not fully learned yet
  - The number buttons tells the system how many additional new cards you want to learn
- Now you learn all questions and answers. After being done with the session, you get a short statistic and a finish button. The finish button brings you back to the normal menu
- The basic algorithm behind it is space repetition, if you are aware of a simple other algorithm feel free to use that.

The basic data structure is a json file with question, potential answers and the right answer. The datastructure for storing the state should be optimized with proper map access and not just always doing some for loop for searching something.

