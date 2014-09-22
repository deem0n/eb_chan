// Generate test of channel operations and simple selects.
// The output of this program is compiled and run to do the
// actual test.

// Each test does only one real send or receive at a time, but phrased
// in various ways that the compiler may or may not rewrite
// into simpler expressions.

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"text/template"
)

func main() {
	out := bufio.NewWriter(os.Stdout)
	fmt.Fprintln(out, header)
	a := new(arg)

	// Generate each kind of test as a separate function to avoid
	// hitting the 6g optimizer with one enormous function.
	// If we name all the functions init we don't have to
	// maintain a list of which ones to run.
	do := func(t *template.Template) {
		fmt.Fprintf(out, `{`)
		for ; next(); a.reset() {
			run(t, a, out)
		}
		fmt.Fprintln(out, `}`)
	}

    // do(recv)
    // do(send)
    do(recvOrder)
	// do(sendOrder)
	// do(nonblock)
    
    fmt.Fprintln(out, footer)
    
	fmt.Fprintln(out, "//", a.nreset, "cases")
	out.Flush()
}

func run(t *template.Template, a interface{}, out io.Writer) {
	if err := t.Execute(out, a); err != nil {
		panic(err)
	}
}

type arg struct {
	def	   bool
	nreset int
}

func (a *arg) Maybe() bool {
	return maybe()
}

func (a *arg) MaybeDefault() bool {
	if a.def {
		return false
	}
	a.def = maybe()
	return a.def
}

func (a *arg) MustDefault() bool {
	return !a.def
}

func (a *arg) reset() {
	a.def = false
	a.nreset++
}

const header = `// GENERATED BY select5c.go

#include "testglue.h"

// channel is buffered so test is single-goroutine.
// we are not interested in the concurrency aspects
// of select, just testing that the right calls happen.
eb_chan c = NULL;
eb_chan nilch = NULL;
int n = 1;
int x = 0;
const void *tmp = NULL;
const void *i = NULL;
eb_chan dummy = NULL;
eb_nsec timeout_dur = eb_nsec_zero;

#define M_SIZE 256
int m[M_SIZE];
int order = 0;

int *f(int *p) {
	return p;
}

// check order of operations by ensuring that
// successive calls to checkorder have increasing o values.
void checkorder(int o) {
	assert(o>order);
	order = o;
}

eb_chan fc(eb_chan c, int o) {
	checkorder(o);
	return c;
}

int *fp(int *p, int o) {
	checkorder(o);
	return p;
}

int fn(int n, int o) {
	checkorder(o);
	return n;
}

void die(int x) {
	abort();
}

int main() {
    c = eb_chan_create(1);
    dummy = eb_chan_create(0);
`

const footer = `
	return 0;
}
`

func parse(name, s string) *template.Template {
	t, err := template.New(name).Parse(s)
	if err != nil {
		panic(fmt.Sprintf("%q: %s", name, err))
	}
	return t
}

var recv = parse("recv", `
	{{/*  Send n, receive it one way or another into x, check that they match. */}}
    eb_chan_send(c, (void*)(intptr_t)n);
    
	{{if .Maybe}}
        assert(eb_chan_recv(c, &tmp));
        x = (int)(intptr_t)tmp;
	{{else}}
        {{if .Maybe}}
            timeout_dur = eb_nsec_zero;
        {{else}}
            timeout_dur = eb_nsec_forever;
        {{end}}
        
    	{
        	{{/*  Receive from c.  Different cases are direct, indirect, :=, interface, and map assignment. */}}
            eb_chan_op op1 = eb_chan_op_recv(c);
            eb_chan_op op2 = eb_chan_op_recv(c);
            eb_chan_op op3 = eb_chan_op_recv(c);
            eb_chan_op op4 = eb_chan_op_recv(c);
            eb_chan_op op5 = eb_chan_op_recv(c);
            eb_chan_op op6 = eb_chan_op_send(dummy, (void*)1);
            eb_chan_op op7 = eb_chan_op_recv(dummy);
            eb_chan_op op8 = eb_chan_op_send(nilch, (void*)1);
            eb_chan_op op9 = eb_chan_op_recv(nilch);
            
            eb_chan_op *r = eb_chan_do(timeout_dur,
                {{if .Maybe}}
            	    &op1,
            	{{else}}
                    {{if .Maybe}}
                	    &op2,
                	{{else}}
                        {{if .Maybe}}
                        	&op3,
                    	{{else}}
                            {{if .Maybe}}
                            	&op4,
                        	{{else}}
                            	&op5,
                        	{{end}}
                        {{end}}
                    {{end}}
                {{end}}
                
            	{{/*  Dummy send, receive to keep compiler from optimizing select. */}}
            	{{if .Maybe}}
                	&op6,
            	{{end}}
            	{{if .Maybe}}
                	&op7,
            	{{end}}
                
            	{{/*  Nil channel send, receive to keep compiler from optimizing select. */}}
            	{{if .Maybe}}
                	&op8,
            	{{end}}
            	{{if .Maybe}}
                	&op9,
            	{{end}}
            );
            
            if (r == &op1) {
                x = (int)(intptr_t)r->val;
            } else if (r == &op2) {
                *f(&x) = (int)(intptr_t)r->val;
            } else if (r == &op3) {
                int y = (int)(intptr_t)r->val;
                x = y;
            } else if (r == &op4) {
                i = r->val;
                x = (int)(intptr_t)i;
            } else if (r == &op5) {
                m[13] = (int)(intptr_t)r->val;
                x = m[13];
            } else if (r == &op6) {
                abort();
            } else if (r == &op7) {
                abort();
            } else if (r == &op8) {
                abort();
            } else if (r == &op9) {
                abort();
            } else {
                assert(timeout_dur == eb_nsec_zero);
            }
    	}
	{{end}}
    
    assert(x == n);
    
	n++;
`)

var send = parse("send", `
	{{/*  Send n one way or another, receive it into x, check that they match. */}}
	{{if .Maybe}}
        eb_chan_send(c, (void*)(intptr_t)n);
	{{else}}
        {{if .Maybe}}
            timeout_dur = eb_nsec_zero;
        {{else}}
            timeout_dur = eb_nsec_forever;
        {{end}}
        
    	{
            eb_chan_op op1 = eb_chan_op_send(c, (void*)(intptr_t)n);
            eb_chan_op op2 = eb_chan_op_send(dummy, (void*)1);
            eb_chan_op op3 = eb_chan_op_recv(dummy);
            eb_chan_op op4 = eb_chan_op_send(nilch, (void*)1);
            eb_chan_op op5 = eb_chan_op_recv(nilch);
            
            eb_chan_op *r = eb_chan_do(timeout_dur,
            	{{/*  Send c <- n.	No real special cases here, because no values come back */}}
            	{{/*  from the send operation. */}}
                &op1,
                
                {{/*  Dummy send, receive to keep compiler from optimizing select. */}}
            	{{if .Maybe}}
                	&op2,
            	{{end}}
        	    
                {{if .Maybe}}
                	&op3,
            	{{end}}
        	    
                {{/*  Nil channel send, receive to keep compiler from optimizing select. */}}
            	{{if .Maybe}}
                	&op4,
            	{{end}}
                
            	{{if .Maybe}}
                	&op5,
            	{{end}}
            );
            
            if (r == &op1) {
            } else if (r == &op2) {
            } else if (r == &op3) {
            } else if (r == &op4) {
            } else if (r == &op5) {
            } else {
                assert(timeout_dur == eb_nsec_zero);
            }
	    }
	{{end}}
    
    assert(eb_chan_recv(c, &tmp));
    x = (int)(intptr_t)(tmp);
	
    assert(x == n);
    
	n++;
`)

var recvOrder = parse("recvOrder", `
	{{/*  Send n, receive it one way or another into x, check that they match. */}}
	{{/*  Check order of operations along the way by calling functions that check */}}
	{{/*  that the argument sequence is strictly increasing. */}}
	order = 0;
    eb_chan_send(c, (void*)(intptr_t)n);
    
	{{if .Maybe}}
    	{{/*  Outside of select, left-to-right rule applies. */}}
    	{{/*  (Inside select, assignment waits until case is chosen, */}}
    	{{/*  so right hand side happens before anything on left hand side. */}}
        {
            int *fpr = fp(&x, 1);
            assert(eb_chan_recv(fc(c, 2), &tmp));
        	*fpr = (int)(intptr_t)tmp;
        }
	{{else}}
        {{if .Maybe}}
            {
                int *fpr = &m[fn(13, 1)];
                assert(eb_chan_recv(fc(c, 2), &tmp));
            	*fpr = (int)(intptr_t)tmp;
            	x = m[13];
            }
    	{{else}}
            {{if .Maybe}}
                timeout_dur = eb_nsec_zero;
            {{else}}
                timeout_dur = eb_nsec_forever;
            {{end}}
            
        	{
                eb_chan_op op1 = eb_chan_op_recv(fc(c, 1));
                eb_chan_op op2 = eb_chan_op_recv(fc(c, 2));
                eb_chan_op op3 = eb_chan_op_recv(fc(c, 3));
                eb_chan_op op4 = eb_chan_op_recv(fc(c, 4));
                eb_chan_op op5 = eb_chan_op_send(fc(dummy, 5), (void*)(intptr_t)fn(1, 6));
                eb_chan_op op6 = eb_chan_op_recv(fc(dummy, 7));
                eb_chan_op op7 = eb_chan_op_send(fc(nilch, 8), (void*)(intptr_t)fn(1, 9));
                eb_chan_op op8 = eb_chan_op_recv(fc(nilch, 10));
                
                eb_chan_op *r = eb_chan_do(timeout_dur,
                	{{/*  Receive from c.  Different cases are direct, indirect, :=, interface, and map assignment. */}}
                	{{if .Maybe}}
                        &op1,
                	{{else}}
                        {{if .Maybe}}
                    	    &op2,
                    	{{else}}
                            {{if .Maybe}}
                                &op3,
                        	{{else}}
                                &op4,
                        	{{end}}
                        {{end}}
                    {{end}}
                
                	{{/*  Dummy send, receive to keep compiler from optimizing select. */}}
                	{{if .Maybe}}
                        &op5,
                	{{end}}
                
                	{{if .Maybe}}
                        &op6,
                	{{end}}
            	
                    {{/*  Nil channel send, receive to keep compiler from optimizing select. */}}
                	{{if .Maybe}}
                        &op7,
                	{{end}}
                
                	{{if .Maybe}}
                        &op8,
                	{{end}}
                );
                
                if (r == &op1) {
                    *fp(&x, 100) = (int)(intptr_t)r->val;
                } else if (r == &op2) {
                    int y = (int)(intptr_t)r->val;
                    x = y;
                } else if (r == &op3) {
                    i = r->val;
                    x = (int)(intptr_t)i;
                } else if (r == &op4) {
                    m[fn(13, 100)] = (int)(intptr_t)r->val;
                    x = m[13];
                } else if (r == &op5) {
                	abort();
                } else if (r == &op6) {
                	abort();
                } else if (r == &op7) {
                	abort();
                } else if (r == &op8) {
                	abort();
                } else {
                    assert(timeout_dur == eb_nsec_zero);
                }
        	}
    	{{end}}
    {{end}}
    
	assert(x == n);
    
	n++;
`)

var sendOrder = parse("sendOrder", `
	{{/*  Send n one way or another, receive it into x, check that they match. */}}
	{{/*  Check order of operations along the way by calling functions that check */}}
	{{/*  that the argument sequence is strictly increasing. */}}
	order = 0
	{{if .Maybe}}
	fc(c, 1) <- fn(n, 2)
	{{else}}
	select {
	{{/*  Blocking or non-blocking, before the receive (same reason as in recv). */}}
	{{if .MaybeDefault}}
	default:
		panic("nonblock")
	{{end}}
	{{/*  Send c <- n.	No real special cases here, because no values come back */}}
	{{/*  from the send operation. */}}
	case fc(c, 1) <- fn(n, 2):
	{{/*  Blocking or non-blocking. */}}
	{{if .MaybeDefault}}
	default:
		panic("nonblock")
	{{end}}
	{{/*  Dummy send, receive to keep compiler from optimizing select. */}}
	{{if .Maybe}}
	case fc(dummy, 3) <- fn(1, 4):
		panic("dummy send")
	{{end}}
	{{if .Maybe}}
	case <-fc(dummy, 5):
		panic("dummy receive")
	{{end}}
	{{/*  Nil channel send, receive to keep compiler from optimizing select. */}}
	{{if .Maybe}}
	case fc(nilch, 6) <- fn(1, 7):
		panic("nilch send")
	{{end}}
	{{if .Maybe}}
	case <-fc(nilch, 8):
		panic("nilch recv")
	{{end}}
	}
	{{end}}
	x = <-c
	if x != n {
		die(x)
	}
	n++
`)

var nonblock = parse("nonblock", `
	x = n
	{{/*  Test various combinations of non-blocking operations. */}}
	{{/*  Receive assignments must not edit or even attempt to compute the address of the lhs. */}}
	select {
	{{if .MaybeDefault}}
	default:
	{{end}}
	{{if .Maybe}}
	case dummy <- 1:
		panic("dummy <- 1")
	{{end}}
	{{if .Maybe}}
	case nilch <- 1:
		panic("nilch <- 1")
	{{end}}
	{{if .Maybe}}
	case <-dummy:
		panic("<-dummy")
	{{end}}
	{{if .Maybe}}
	case x = <-dummy:
		panic("<-dummy x")
	{{end}}
	{{if .Maybe}}
	case **(**int)(nil) = <-dummy:
		panic("<-dummy (and didn't crash saving result!)")
	{{end}}
	{{if .Maybe}}
	case <-nilch:
		panic("<-nilch")
	{{end}}
	{{if .Maybe}}
	case x = <-nilch:
		panic("<-nilch x")
	{{end}}
	{{if .Maybe}}
	case **(**int)(nil) = <-nilch:
		panic("<-nilch (and didn't crash saving result!)")
	{{end}}
	{{if .MustDefault}}
	default:
	{{end}}
	}
	if x != n {
		die(x)
	}
	n++
`)

// Code for enumerating all possible paths through
// some logic.	The logic should call choose(n) when
// it wants to choose between n possibilities.
// On successive runs through the logic, choose(n)
// will return 0, 1, ..., n-1.	The helper maybe() is
// similar but returns true and then false.
//
// Given a function gen that generates an output
// using choose and maybe, code can generate all
// possible outputs using
//
//	for next() {
//		gen()
//	}

type choice struct {
	i, n int
}

var choices []choice
var cp int = -1

func maybe() bool {
	return choose(2) == 0
}

func choose(n int) int {
	if cp >= len(choices) {
		// never asked this before: start with 0.
		choices = append(choices, choice{0, n})
		cp = len(choices)
		return 0
	}
	// otherwise give recorded answer
	if n != choices[cp].n {
		panic("inconsistent choices")
	}
	i := choices[cp].i
	cp++
	return i
}

func next() bool {
	if cp < 0 {
		// start a new round
		cp = 0
		return true
	}

	// increment last choice sequence
	cp = len(choices) - 1
	for cp >= 0 && choices[cp].i == choices[cp].n-1 {
		cp--
	}
	if cp < 0 {
		choices = choices[:0]
		return false
	}
	choices[cp].i++
	choices = choices[:cp+1]
	cp = 0
	return true
}