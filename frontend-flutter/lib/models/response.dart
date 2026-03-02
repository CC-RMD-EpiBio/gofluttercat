class ItemResponse {
  final String itemName;
  final int value;

  ItemResponse({required this.itemName, required this.value});

  Map<String, dynamic> toJson() {
    return {
      'item_name': itemName,
      'value': value,
    };
  }

  factory ItemResponse.fromJson(Map<String, dynamic> json) {
    return ItemResponse(
      itemName: json['item_name'] as String,
      value: (json['value'] as num).toInt(),
    );
  }
}
