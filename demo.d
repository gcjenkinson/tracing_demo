opentracing*:::start
{
	self->sc = copyinstr(arg1);
}

opentracing*:::finish
{
	self->sc = 0;
}

syscall::write:entry
/self->sc != 0/
{
	ustack();
}

