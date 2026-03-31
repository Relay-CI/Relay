import React from "react";
import { cn } from "../../lib/utils";

const Input = React.forwardRef(function Input({ className, ...props }, ref) {
  return <input ref={ref} className={cn("ui-input text-input", className)} {...props} />;
});

export { Input };
