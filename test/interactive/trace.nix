{ lib, pkgs, ... }:
let
  tracing = name: "\"/sys/kernel/debug/tracing/${name}\"";
in
{
  environment.systemPackages = [
    (pkgs.writeShellScriptBin "hakurei-set-up-tracing" ''
      set -e
      echo "$1" > ${tracing "set_graph_function"}
      echo function_graph > ${tracing "current_tracer"}
      echo funcgraph-tail > ${tracing "trace_options"}
      echo funcgraph-retval > ${tracing "trace_options"}
      echo nofuncgraph-cpu > ${tracing "trace_options"}
      echo nofuncgraph-overhead > ${tracing "trace_options"}
      echo nofuncgraph-duration > ${tracing "trace_options"}
    '')
    (pkgs.writeShellScriptBin "hakurei-print-trace" "exec cat ${tracing "trace"}")
    (pkgs.writeShellScriptBin "hakurei-consume-trace" "exec cat ${tracing "trace_pipe"}")
  ];

  boot.kernelPatches = [
    {
      name = "funcgraph-retval";
      patch = null;
      extraStructuredConfig.FUNCTION_GRAPH_RETVAL = lib.kernel.yes;
    }
  ];
}
