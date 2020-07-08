opentracing*:::start
{
	self->sc = copyinstr(arg1);
	trace(self->sc);
}

opentracing*:::finish
{
	self->sc = 0;
}

syscall::write:entry
/self->sc != 0/
{
	trace(self->sc);
}

syscall::write:return
/self->sc != 0/
{
	trace(self->sc);
}
